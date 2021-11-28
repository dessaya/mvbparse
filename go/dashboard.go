package mvb

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
)

const (
	portPageSize = 16
	maxPort      = 1<<12 - portPageSize
)

var initialPort uint16 = 0

func initDashboardFlags() {
	flag.Func("port", "initial port offset", func(s string) (err error) {
		initialPort, err = decodePort(s)
		return
	})
}

func decodePort(s string) (uint16, error) {
	n, err := strconv.ParseUint(s, 16, 12)
	if err != nil {
		return 0, err
	}
	if n > maxPort {
		return maxPort, nil
	}
	return uint16(n), nil
}

type CaptureMode bool

const (
	CaptureModeTelegrams = false
	CaptureModeVars      = true
)

type Dashboard struct {
	screen        tcell.Screen
	stats         Stats
	port          uint16
	captureMode   CaptureMode
	captureOffset int
	portFilter    *portFilter
	paused        bool
	n             func() uint64
}

func NewDashboard(n func() uint64) *Dashboard {
	return &Dashboard{
		stats: NewStats(),
		port:  uint16(initialPort),
		n:     n,
	}
}

var (
	defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorWhite)
	errStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorRed)
)

func (d *Dashboard) init() {
	tty, err := tcell.NewDevTty()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	s, err := tcell.NewTerminfoScreenFromTty(tty)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(defStyle)
	d.screen = s
}

func (d *Dashboard) quit() {
	d.screen.Fini()
	os.Exit(0)
}

const screenWidth = 80

func (d *Dashboard) render() {
	s := d.screen
	s.Clear()
	switch {
	case d.stats.Capture != nil:
		d.renderCapture(d.stats.Capture)
	default:
		d.renderMain()
	}
	s.Show()
}

func (d *Dashboard) showPort(y int, port uint16) {
	v := d.stats.Vars[port]
	drawTextLine(d.screen, 1, y, screenWidth, defStyle, fmt.Sprintf("port %03x = %s", port, hex.EncodeToString(v)))
}

func (d *Dashboard) renderMain() {
	s := d.screen
	y := 0
	drawTextLine(s, 1, y, screenWidth, defStyle, fmt.Sprintf("Total: %d telegrams", d.stats.Total))
	drawTextLine(s, 40, y, screenWidth, defStyle, fmt.Sprintf("%.3fs", sampleTimestamp(d.n()).Seconds()))
	y++

	drawHLine(s, y, screenWidth, defStyle)
	y++

	rate := d.stats.Rate()
	drawTextLine(s, 1, y, screenWidth, defStyle, fmt.Sprintf(
		"%s %6d telegrams/s",
		spark(rate),
		rate[len(rate)-1],
	))
	y++

	drawHLine(s, y, screenWidth, defStyle)
	y++

	for i := range d.stats.mrRates {
		rate := d.stats.MRRate(MasterRequest(i))
		drawTextLine(s, 1, y, screenWidth, defStyle, fmt.Sprintf(
			"%s %6d telegrams/s %s",
			spark(rate),
			rate[len(rate)-1],
			MasterRequest(i),
		))
		y++
	}

	drawHLine(s, y, screenWidth, defStyle)
	y++

	if d.portFilter != nil {
		d.showPort(y, d.portFilter.port())
		y++
	} else {
		for port := d.port; port < d.port+portPageSize; port++ {
			d.showPort(y, port)
			y++
		}
	}

	drawHLine(s, y, screenWidth, defStyle)
	y++

	errorRate := d.stats.ErrorRate()
	drawTextLine(s, 1, y, screenWidth, errStyle, fmt.Sprintf(
		"%s %6d errors/s",
		spark(errorRate),
		errorRate[len(errorRate)-1],
	))
	y++

	for i := 0; i < len(d.stats.ErrorLog); i++ {
		err := d.stats.ErrorLog[i]
		drawTextLine(s, 1, y, screenWidth, errStyle, fmt.Sprintf(
			"[%s] %s",
			sampleTimestamp(err.N()),
			err.Error(),
		))
		y++
		if annotate {
			trace := traceSamples(err.samples)
			drawTextLine(s, 1, y, len(trace)+1, errStyle, fmt.Sprintf(
				"%s",
				trace,
			))
			y++
		}
	}

	s.Show()
}

func (d *Dashboard) renderCapture(c *Capture) {
	switch d.captureMode {
	case CaptureModeTelegrams:
		d.renderCaptureTelegrams(c)
	case CaptureModeVars:
		d.renderCaptureVars(c)
	}
}

func (d *Dashboard) renderCaptureTelegrams(c *Capture) {
	y := -d.captureOffset
	for _, t := range c.Telegrams {
		s := t.String()
		drawTextLine(d.screen, 1, y, len(s)+1, defStyle, s)
		y++
	}
}

func (d *Dashboard) renderCaptureVars(c *Capture) {
	y := -d.captureOffset
	if d.portFilter != nil {
		d.renderPortCapture(y, d.portFilter.port())
	} else {
		for _, port := range c.SeenPorts {
			y = d.renderPortCapture(y, uint16(port))
		}
	}
}

func (d *Dashboard) renderPortCapture(y int, port uint16) int {
	drawTextLine(d.screen, 1, y, screenWidth, defStyle, fmt.Sprintf("port %03x", port))
	y++
	for _, change := range d.stats.Capture.Vars[port] {
		drawTextLine(d.screen, 1, y, screenWidth, defStyle, fmt.Sprintf(
			"  [%s] %x",
			sampleTimestamp(change.N),
			hex.EncodeToString(change.Value),
		))
		y++
	}
	return y
}

func addPortOffset(port uint16, d int) uint16 {
	if int(port)+d < 0 {
		return 0
	}
	if int(port)+d > maxPort {
		return maxPort
	}
	return port + uint16(d)
}

func addCaptureOffset(o int, d int) int {
	if o+d < 0 {
		return 0
	}
	return o + d
}

func (d *Dashboard) Loop(mvbEvents chan Event) {
	d.init()

	tcellEvents := make(chan tcell.Event)
	go d.screen.ChannelEvents(tcellEvents, make(chan struct{}))

	renderTicker := time.Tick(100 * time.Millisecond)
	secondsTicker := time.Tick(1 * time.Second)
	dirty := false

	d.render()
	for {
		select {
		case <-renderTicker:
			if !d.paused && dirty {
				d.render()
			}

		case <-secondsTicker:
			d.stats.Tick()
			dirty = true

		case ev := <-tcellEvents:
			switch ev := ev.(type) {
			case *tcell.EventResize:
				d.screen.Sync()
			case *tcell.EventKey:
				switch {
				case ev.Key() == tcell.KeyCtrlC || ev.Rune() == 'q' || ev.Rune() == 'Q':
					d.quit()
				case ev.Rune() == 'm' || ev.Rune() == 'M':
					d.captureMode = !d.captureMode
					d.captureOffset = 0
				case ev.Rune() == 'p' || ev.Rune() == 'P':
					d.paused = !d.paused
				case ev.Rune() == ' ':
					d.stats.StartStopCapture()
					d.captureOffset = 0
				case ev.Key() == tcell.KeyESC:
					if d.portFilter != nil {
						d.portFilter = nil
					} else {
						d.stats.DiscardCapture()
					}
				case ev.Key() == tcell.KeyCtrlL:
					d.screen.Sync()
				case d.tryScroll(ev):
				case d.tryPortFilter(ev):
					d.captureOffset = 0
				}
			}

		case ev := <-mvbEvents:
			switch ev := ev.(type) {
			case *Telegram:
				d.stats.CountTelegram(ev)
			case Error:
				d.stats.CountError(ev)
			}
			dirty = true
		}
	}
}

func (d *Dashboard) tryScroll(ev *tcell.EventKey) bool {
	switch {
	case d.stats.Capture != nil:
		return d.tryScrollCapture(ev)
	default:
		return d.tryScrollPort(ev)
	}
}

func (d *Dashboard) tryScrollPort(ev *tcell.EventKey) bool {
	switch {
	case ev.Rune() == '+':
		d.port = addPortOffset(d.port, portPageSize*16)
	case ev.Key() == tcell.KeyPgDn:
		d.port = addPortOffset(d.port, portPageSize)
	case ev.Rune() == '-':
		d.port = addPortOffset(d.port, -portPageSize*16)
	case ev.Key() == tcell.KeyPgUp:
		d.port = addPortOffset(d.port, -portPageSize)
	case ev.Key() == tcell.KeyDown:
		d.port = addPortOffset(d.port, 1)
	case ev.Key() == tcell.KeyUp:
		d.port = addPortOffset(d.port, -1)
	case ev.Key() == tcell.KeyHome:
		d.port = 0
	case ev.Key() == tcell.KeyEnd:
		d.port = maxPort
	default:
		return false
	}
	return true
}

func (d *Dashboard) tryScrollCapture(ev *tcell.EventKey) bool {
	switch {
	case ev.Key() == tcell.KeyPgDn:
		d.captureOffset = addCaptureOffset(d.captureOffset, 10)
	case ev.Key() == tcell.KeyPgUp:
		d.captureOffset = addCaptureOffset(d.captureOffset, -10)
	case ev.Key() == tcell.KeyDown:
		d.captureOffset = addCaptureOffset(d.captureOffset, 1)
	case ev.Key() == tcell.KeyUp:
		d.captureOffset = addCaptureOffset(d.captureOffset, -1)
	case ev.Key() == tcell.KeyHome:
		d.captureOffset = 0
	default:
		return false
	}
	return true
}

type portFilter struct {
	hex string
}

func (p *portFilter) port() uint16 {
	n, err := strconv.ParseInt(p.hex, 16, 12)
	if err != nil {
		return 0
	}
	return uint16(n)
}

func (d *Dashboard) tryPortFilter(ev *tcell.EventKey) bool {
	if ev.Rune() == '/' {
		d.portFilter = &portFilter{}
		return true
	}
	pf := d.portFilter
	if pf == nil {
		return false
	}
	s := string(ev.Rune())
	_, err := strconv.ParseInt(s, 16, 8)
	if err != nil {
		return false
	}
	pf.hex += s
	if len(pf.hex) > 3 {
		pf.hex = pf.hex[1:]
	}
	return true
}

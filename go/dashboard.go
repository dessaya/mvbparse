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
	screen             tcell.Screen
	stats              Stats
	port               uint16
	captureMode        CaptureMode
	captureOffset      int
	portFilter         *portFilter
	paused             bool
	n                  func() uint64
	watchedPorts       []RecorderPortSpec
	watchedPortsOffset int
}

func NewDashboard(n func() uint64, watchedPorts []RecorderPortSpec) *Dashboard {
	return &Dashboard{
		stats:        NewStats(),
		port:         uint16(initialPort),
		n:            n,
		watchedPorts: watchedPorts,
	}
}

var (
	defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorWhite)
	invStyle = defStyle.Reverse(true)
	errStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorRed)
	capStyle = errStyle.Background(tcell.ColorWhite).Reverse(true).Bold(true)
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

func (d *Dashboard) render() {
	if d.paused {
		return
	}
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
	drawText(d.screen, 0, y, defStyle, fmt.Sprintf("port %03x = %s", port, hex.EncodeToString(v)))
}

func (d *Dashboard) renderHeader(style tcell.Style, s string) {
	w, _ := d.screen.Size()
	drawText(d.screen, 0, 0, style, fmt.Sprintf(fmt.Sprintf("%%-%ds", w), s))
}

func (d *Dashboard) renderMain() {
	s := d.screen

	d.renderHeader(invStyle, "MVB [space: capture] [/: port filter] [p: pause] [q: quit]")
	y := 1

	drawText(s, 0, y, defStyle, fmt.Sprintf("Total: %d telegrams", d.stats.Total))
	drawText(s, 40, y, defStyle, fmt.Sprintf("%.3fs", sampleTimestamp(d.n()).Seconds()))
	y++

	drawHLine(s, y, defStyle)
	y++

	rate := d.stats.Rate()
	drawText(s, 0, y, defStyle, fmt.Sprintf(
		"%s %6d telegrams/s",
		spark(rate),
		rate[len(rate)-1],
	))
	y++

	drawHLine(s, y, defStyle)
	y++

	if len(d.watchedPorts) == 0 {
		for i := range d.stats.mrRates {
			rate := d.stats.MRRate(MasterRequest(i))
			drawText(s, 0, y, defStyle, fmt.Sprintf(
				"%s %6d telegrams/s %s",
				spark(rate),
				rate[len(rate)-1],
				MasterRequest(i),
			))
			y++
		}

		drawHLine(s, y, defStyle)
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
	} else {
		y = d.renderWatchedPorts(y)
	}

	drawHLine(s, y, defStyle)
	y++

	errorRate := d.stats.ErrorRate()
	drawText(s, 0, y, errStyle, fmt.Sprintf(
		"%s %6d errors/s",
		spark(errorRate),
		errorRate[len(errorRate)-1],
	))
	y++

	for i := 0; i < len(d.stats.ErrorLog); i++ {
		err := d.stats.ErrorLog[i]
		drawText(s, 0, y, errStyle, fmt.Sprintf(
			"[%.3fs] %s",
			sampleTimestamp(err.N()).Seconds(),
			err.Error(),
		))
		y++
		if annotate {
			trace := traceSamples(err.samples)
			drawText(s, 0, y, errStyle, fmt.Sprintf("%s", trace))
			y++
		}
	}

	s.Show()
}

func (d *Dashboard) renderWatchedPorts(y int) int {
	s := d.screen
	for _, w := range d.watchedPorts[d.watchedPortsOffset:] {
		drawText(s, 0, y, defStyle, fmt.Sprintf("%32s %x", w.Desc, slice(d.stats.Vars[w.Port], w.I, w.J)))
		y++
	}
	return y
}

func (d *Dashboard) renderCapture(c *Capture) {
	switch d.captureMode {
	case CaptureModeTelegrams:
		d.renderCaptureTelegrams(c)
	case CaptureModeVars:
		d.renderCaptureVars(c)
	}
}

func (d *Dashboard) scrollCaptureTelegramsEnd(c *Capture) {
	_, h := d.screen.Size()
	d.captureOffset = addCaptureOffset(0, len(c.Telegrams)-h+1)
}

func (d *Dashboard) renderCaptureTelegrams(c *Capture) {
	if c.Stopped {
		d.renderHeader(invStyle, "CAPTURE (stopped) [m: show vars] [esc: back]")
	} else {
		d.renderHeader(capStyle, "CAPTURE (running) [space: stop]")
	}

	if d.captureOffset >= len(c.Telegrams) {
		return
	}
	_, h := d.screen.Size()
	y := 1
	for _, t := range c.Telegrams[d.captureOffset:] {
		drawText(d.screen, 0, y, defStyle, t.String())
		y++
		if y > h {
			return
		}
	}
}

func (d *Dashboard) renderCaptureVars(c *Capture) {
	if c.Stopped {
		d.renderHeader(invStyle, "CAPTURE (stopped) [m: show telegrams] [/: port filter] [esc: back]")
	} else {
		d.renderHeader(capStyle, "CAPTURE (running) [space: stop]")
	}

	_, h := d.screen.Size()
	y := 1 - d.captureOffset
	if d.portFilter != nil {
		d.renderPortCapture(y, h, d.portFilter.port())
	} else {
		for _, port := range c.SeenPorts {
			y = d.renderPortCapture(y, h, uint16(port))
		}
	}
}

func (d *Dashboard) renderPortCapture(y int, h int, port uint16) int {
	if y > 0 && y < h {
		drawText(d.screen, 0, y, defStyle, fmt.Sprintf("port %03x", port))
	}
	y++
	for _, change := range d.stats.Capture.Vars[port] {
		if y > 0 {
			drawText(d.screen, 0, y, defStyle, fmt.Sprintf(
				"  [%.3fs] %s",
				sampleTimestamp(change.N).Seconds(),
				hex.EncodeToString(change.Value),
			))
		}
		y++
		if y > h {
			return y
		}
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

func addWatchedPortOffset(o int, d int, max int) int {
	if o+d < 0 {
		return 0
	}
	if o+d > max {
		return max
	}
	return o + d
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
			if dirty {
				d.render()
				dirty = false
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
					if d.stats.Capture != nil && !d.stats.Capture.Stopped {
						d.captureOffset = 0
					}
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
			d.render()

		case ev := <-mvbEvents:
			switch ev := ev.(type) {
			case *Telegram:
				d.stats.CountTelegram(ev)
				c := d.stats.Capture
				if c != nil && !c.Stopped && d.captureMode == CaptureModeTelegrams {
					d.scrollCaptureTelegramsEnd(c)
				}
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
	case len(d.watchedPorts) != 0:
		return d.tryScrollWatchedPorts(ev)
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

func (d *Dashboard) tryScrollWatchedPorts(ev *tcell.EventKey) bool {
	switch {
	case ev.Key() == tcell.KeyPgDn:
		d.watchedPortsOffset = addWatchedPortOffset(d.watchedPortsOffset, portPageSize, len(d.watchedPorts)-1)
	case ev.Key() == tcell.KeyPgUp:
		d.watchedPortsOffset = addWatchedPortOffset(d.watchedPortsOffset, -portPageSize, len(d.watchedPorts)-1)
	case ev.Key() == tcell.KeyDown:
		d.watchedPortsOffset = addWatchedPortOffset(d.watchedPortsOffset, 1, len(d.watchedPorts)-1)
	case ev.Key() == tcell.KeyUp:
		d.watchedPortsOffset = addWatchedPortOffset(d.watchedPortsOffset, -1, len(d.watchedPorts)-1)
	case ev.Key() == tcell.KeyHome:
		d.watchedPortsOffset = 0
	case ev.Key() == tcell.KeyEnd:
		d.watchedPortsOffset = len(d.watchedPorts) - 1
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
	case ev.Key() == tcell.KeyEnd:
		if d.captureMode == CaptureModeTelegrams {
			d.scrollCaptureTelegramsEnd(d.stats.Capture)
		}
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

package mvb

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

type Dashboard struct {
	screen tcell.Screen
	stats  Stats
}

func NewDashboard() *Dashboard {
	return &Dashboard{
		stats: NewStats(),
	}
}

var (
	defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorWhite)
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
	s := d.screen
	s.Clear()

	w := 80
	y := 1
	drawTextLine(s, 1, y, w, defStyle, fmt.Sprintf("Total: %d telegrams", d.stats.Total))
	y++

	rate := d.stats.Rate()
	drawTextLine(s, 1, y, w, defStyle, fmt.Sprintf(
		"%s %d telegrams/s",
		spark(rate),
		rate[len(rate)-1],
	))
	y++
	drawTextLine(s, 1, y, w, defStyle, strings.Repeat(string(tcell.RuneHLine), w))
	y++

	for _, err := range d.stats.Errors {
		drawTextLine(s, 1, y, w, defStyle, fmt.Sprintf("[%s] %s", sampleTimestamp(err.N()), err.Error()))
		y++
	}

	s.Show()
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
				case ev.Key() == tcell.KeyCtrlL:
					d.screen.Sync()
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

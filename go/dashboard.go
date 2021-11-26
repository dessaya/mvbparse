package mvb

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

type Stats struct {
	N      uint64
	errors []Error
}

type Dashboard struct {
	screen tcell.Screen
	stats  Stats
}

func NewDashboard() *Dashboard {
	return &Dashboard{
		stats: Stats{
			errors: make([]Error, 0, 10),
		},
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
	drawTextLine(s, 1, 1, w, defStyle, fmt.Sprintf("%d", d.stats.N))
	drawTextLine(s, 1, 2, w, defStyle, strings.Repeat(string(tcell.RuneHLine), w))

	for i, err := range d.stats.errors {
		t := time.Duration(float64(uint64(time.Second)*err.N()) / SampleRate)
		drawTextLine(s, 1, 3+i, w, defStyle, fmt.Sprintf("[%s] %s", t, err.Error()))
	}

	s.Show()
}

func (d *Dashboard) Loop(mvbEvents chan Event) {
	d.init()

	tcellEvents := make(chan tcell.Event)
	go d.screen.ChannelEvents(tcellEvents, make(chan struct{}))

	ticker := time.Tick(100 * time.Millisecond)
	dirty := false

	d.render()
	for {
		select {
		case <-ticker:
			if dirty {
				d.render()
				dirty = false
			}

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
				d.stats.N++
				dirty = true
			case Error:
				if len(d.stats.errors) == cap(d.stats.errors) {
					dst := d.stats.errors[:len(d.stats.errors)-1]
					src := d.stats.errors[1:]
					copy(dst, src)
					d.stats.errors = dst
				}
				d.stats.errors = append(d.stats.errors, ev)
				dirty = true
			}
		}
	}
}

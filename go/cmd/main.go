package main

import (
	"fmt"
	"mvb"
	"os"
	"runtime/pprof"
	"time"
)

func main() {
	{
		fp, err := os.Create("pprof.pprof")
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		pprof.StartCPUProfile(fp)
		defer pprof.StopCPUProfile()
	}

	events := make(chan mvb.Event)
	go mvb.NewDecoder(mvb.NewMVBStream(), events).Loop()

	for {
		ev, ok := <-events
		if !ok {
			break
		}
		t := time.Duration(float64(uint64(time.Second)*ev.N()) / mvb.SampleRate)
		switch ev := ev.(type) {
		case *mvb.Telegram:
			fmt.Printf("[%s] %+v\n", t, ev)
		case mvb.Error:
			fmt.Printf("[%s] %s\n", t, ev.Error())
		}
	}
}

package main

import (
	"log"
	"mvb"
	"os"
	"runtime"
	"runtime/pprof"

	"golang.org/x/term"
)

const (
	cpuProfile   = false
	blockProfile = false
)

func main() {
	if cpuProfile {
		fp, err := os.Create("pprof.pprof")
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		pprof.StartCPUProfile(fp)
		defer pprof.StopCPUProfile()
	}

	if blockProfile {
		runtime.SetBlockProfileRate(1)
		defer func() {
			fp, err := os.Create("pprof.pprof")
			if err != nil {
				panic(err)
			}
			defer fp.Close()
			pprof.Lookup("block").WriteTo(fp, 0)
		}()
	}

	log.SetFlags(0)

	if term.IsTerminal(0) {
		log.Fatalf("stdin must be a pipe")
	}

	events := make(chan mvb.Event)
	go mvb.NewDecoder(mvb.NewMVBStream()).Loop(events)
	mvb.NewDashboard().Loop(events)
}

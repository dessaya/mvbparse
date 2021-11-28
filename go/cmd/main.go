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
		fp, err := os.Create("cpu.pprof")
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
			fp, err := os.Create("block.pprof")
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

	mvb.InitFlags()

	events := make(chan mvb.Event)
	decoder := mvb.NewDecoder(mvb.NewMVBStream())
	go decoder.Loop(events)
	mvb.NewDashboard(decoder.N).Loop(events)
}

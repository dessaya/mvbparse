package main

import (
	"flag"
	"log"
	"mvb"
	"os"
)

func usage() {
	log.Fatalf("usage: %s <port>[:i:j] <desc> [<port>[:i:j] <desc> ...]", os.Args[0])
}

func main() {
	log.SetFlags(0)

	mvb.InitFlags()

	ports, err := mvb.ParseRecorderPortSpecs(flag.CommandLine.Args())
	if err != nil {
		usage()
	}

	events := make(chan mvb.Event)
	decoder := mvb.NewDecoder(mvb.NewMVBStream())
	go decoder.Loop(events)

	mvb.NewRecorder(ports).Loop(events)
}

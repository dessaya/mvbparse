package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"log"
	"mvb"
	"os"
	"strconv"
	"strings"
)

func usage() {
	log.Fatalf("usage: %s <port>[:i:j] [<port>[:i:j] ...]", os.Args[0])
}

func main() {
	log.SetFlags(0)

	mvb.InitFlags()

	events := make(chan mvb.Event)
	decoder := mvb.NewDecoder(mvb.NewMVBStream())
	go decoder.Loop(events)

	args := flag.CommandLine.Args()
	if len(args) != 1 {
		usage()
	}

	var ports []mvb.RecorderPortSpec
	for _, arg := range args {
		parts := strings.Split(arg, ":")
		if len(parts) != 1 && len(parts) != 3 {
			usage()
		}

		i, j := -1, -1
		if len(parts) == 3 {
			var err error
			i, err = strconv.Atoi(parts[1])
			if err != nil {
				usage()
			}
			j, err = strconv.Atoi(parts[2])
			if err != nil {
				usage()
			}
		}

		portHex := parts[0]
		if strings.HasPrefix(portHex, "0x") {
			portHex = portHex[2:]
		}
		if len(portHex)%2 != 0 {
			portHex = "0" + portHex
		}
		b, err := hex.DecodeString(portHex)
		if err != nil {
			usage()
		}
		if len(b) > 2 {
			log.Fatalf("port is too high")
		}
		port := binary.BigEndian.Uint16(b)

		ports = append(ports, mvb.RecorderPortSpec{
			Port: port,
			I:    i,
			J:    j,
		})
	}

	mvb.NewRecorder(ports).Loop(events)
}

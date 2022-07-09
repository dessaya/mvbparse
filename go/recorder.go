package mvb

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

type RecorderPortSpec struct {
	Port uint16
	I    int
	J    int
	Desc string
}

func (s *RecorderPortSpec) String() string {
	if s.I == -1 {
		return fmt.Sprintf("%03x-%s", s.Port, slug(s.Desc))
	}
	return fmt.Sprintf("%03x-%d-%d-%s", s.Port, s.I, s.J, slug(s.Desc))
}

func slug(s string) string {
	s = strings.Trim(s, " ")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

type Recorder struct {
	ports []*portRecorder
}

func NewRecorder(ports []RecorderPortSpec) *Recorder {
	r := &Recorder{}
	for _, portSpec := range ports {
		r.ports = append(r.ports, &portRecorder{
			RecorderPortSpec: portSpec,
		})
	}
	return r
}

func (r *Recorder) logError(err Error) {
	log.Println(err.Error())
}

func (r *Recorder) Loop(mvbEvents chan Event) {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	done := false
	for !done {
		select {
		case ev := <-mvbEvents:
			switch t := ev.(type) {
			case *Telegram:
				fcode := fcodes[t.Master.FCode]
				if fcode.MasterRequest == MR_PROCESS_DATA && t.Slave != nil {
					for _, p := range r.ports {
						if t.Master.Address == p.Port {
							p.write(time.Now(), slice(t.Slave.data, p.I, p.J))
						}
					}
				}
			case Error:
				r.logError(t)
			}
		case <-sigint:
			log.Printf("interrupt - quitting...")
			done = true
		}
	}

	for _, p := range r.ports {
		p.close()
	}
}

type portRecorder struct {
	RecorderPortSpec
	fp       *os.File
	date     string
	lastSeen []byte
}

func (r *portRecorder) write(t time.Time, value []byte) {
	now := time.Now().Format("2006-01-02")
	if r.fp == nil || now != r.date {
		r.fp = r.rotate(r.fp, now)
		r.date = now
		r.lastSeen = nil
	}

	if bytes.Equal(r.lastSeen, value) {
		return
	}
	r.lastSeen = value

	ts := t.Format("15:04:05.000")
	if VerboseFlag {
		log.Printf("%s %32s %x\n", ts, &r.RecorderPortSpec, value)
	}
	if _, err := r.fp.Write([]byte(fmt.Sprintf("%s,%x\n", ts, value))); err != nil {
		panic(err)
	}
}

const baseDir = "csv"

func (r *portRecorder) rotate(fp *os.File, date string) *os.File {
	if fp != nil {
		if err := fp.Close(); err != nil {
			panic(err)
		}
	}

	dir := baseDir + "/" + date
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	fp, err = os.OpenFile(
		fmt.Sprintf("%s/%s.csv", dir, &r.RecorderPortSpec),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE,
		0666,
	)
	if err != nil {
		panic(err)
	}
	return fp
}

func (r *portRecorder) close() {
	if err := r.fp.Close(); err != nil {
		panic(err)
	}
}

func slice(v []byte, i, j int) []byte {
	if i < 0 {
		return v
	}
	if i > len(v)-1 || j <= i {
		return nil
	}
	if j > len(v) {
		j = len(v)
	}
	return v[i:j]
}

var portSpecError = fmt.Errorf("invalid recorder port spec")

func ParseRecorderPortSpecs(args []string) ([]RecorderPortSpec, error) {
	if len(args)%2 != 0 {
		return nil, portSpecError
	}

	var ports []RecorderPortSpec

	for i := 0; i < len(args); i += 2 {
		arg, desc := args[i], args[i+1]
		parts := strings.Split(arg, ":")
		if len(parts) != 1 && len(parts) != 3 {
			return nil, portSpecError
		}

		i, j := -1, -1
		if len(parts) == 3 {
			var err error
			i, err = strconv.Atoi(parts[1])
			if err != nil {
				return nil, portSpecError
			}
			j, err = strconv.Atoi(parts[2])
			if err != nil {
				return nil, portSpecError
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
			return nil, portSpecError
		}
		if len(b) > 2 {
			log.Fatalf("port is too high")
		}
		port := binary.BigEndian.Uint16(b)

		ports = append(ports, RecorderPortSpec{
			Port: port,
			I:    i,
			J:    j,
			Desc: desc,
		})
	}
	return ports, nil
}

package mvb

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
)

type RecorderPortSpec struct {
	Port uint16
	I    int
	J    int
}

func (s *RecorderPortSpec) String() string {
	if s.I == -1 {
		return fmt.Sprintf("%03x", s.Port)
	}
	return fmt.Sprintf("%03x:%d:%d", s.Port, s.I, s.J)
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

	if _, err := r.fp.Write([]byte(fmt.Sprintf("%s,%x\n", t.Format("15:04:05.000"), value))); err != nil {
		panic(err)
	}
}

func (r *portRecorder) rotate(fp *os.File, date string) *os.File {
	if fp != nil {
		if err := fp.Close(); err != nil {
			panic(err)
		}
	}

	fp, err := os.OpenFile(
		fmt.Sprintf("mvb-%s-%s.csv", &r.RecorderPortSpec, date),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE,
		0666,
	)
	if err != nil {
		panic(err)
	}
	return fp
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

package mvb

import (
	"bytes"
	"flag"
	"io"
	"os"
	"strconv"
	"time"
)

// samples per second
const SampleRate = 12_000_000

func sampleTimestamp(n uint64) time.Duration {
	return time.Duration(float64(uint64(time.Second)*n) / SampleRate)
}

var (
	signalHigh = byte(0xff)
	signalLow  = byte(0xfe)
	annotate   = false
)

func initInputFlags() {
	flag.Func("high", "byte value for input = high", func(s string) (err error) {
		signalHigh, err = decodeByte(s)
		return
	})
	flag.Func("low", "byte value for input = low", func(s string) (err error) {
		signalLow, err = decodeByte(s)
		return
	})
	flag.BoolVar(&annotate, "annotate", annotate, "activate annotations")
}

func decodeByte(s string) (byte, error) {
	n, err := strconv.ParseUint(s, 16, 8)
	if err != nil {
		return 0, err
	}
	return byte(n), nil
}

const BufSize = SampleRate / 2

type BufferedReader struct {
	ready chan interface{}
	done  chan *buffer

	cur *buffer

	n uint64

	samples *Samples
}

type buffer struct {
	arr [BufSize]byte
	buf []byte
}

func NewDoubleBufferedReader(r io.Reader) *BufferedReader {
	var bufs [2]buffer
	d := &BufferedReader{
		ready:   make(chan interface{}, len(bufs)),
		done:    make(chan *buffer, len(bufs)),
		samples: NewSamples(64),
	}
	for i := range bufs {
		d.done <- &bufs[i]
	}
	go bufferingLoop(r, d.ready, d.done)
	return d
}

func bufferingLoop(r io.Reader, ready chan interface{}, done chan *buffer) {
	for {
		buf := <-done
		n, err := r.Read(buf.arr[:])
		if err != nil {
			ready <- err
			return
		}
		buf.buf = buf.arr[:n]
		ready <- buf
	}
}

func (d *BufferedReader) buffer() error {
	if d.cur == nil {
		x := <-d.ready
		switch x := x.(type) {
		case *buffer:
			d.cur = x
			return nil
		case error:
			return x
		}
	}
	return nil
}

func (d *BufferedReader) disposeBuffer() {
	d.done <- d.cur
	d.cur = nil
}

func (d *BufferedReader) ReadByte() (byte, error) {
	err := d.buffer()
	if err != nil {
		return 0, err
	}
	b := d.cur.buf[0]
	d.cur.buf = d.cur.buf[1:]
	d.n++
	if len(d.cur.buf) == 0 {
		d.disposeBuffer()
	}
	if annotate {
		d.samples.PushByte(b)
	}
	return b, nil
}

func (d *BufferedReader) Discard(remaining int) error {
	for remaining > 0 {
		err := d.buffer()
		if err != nil {
			return err
		}
		n := remaining
		if n > len(d.cur.buf) {
			n = len(d.cur.buf)
		}
		if annotate {
			for _, b := range d.cur.buf[:n] {
				d.samples.PushByte(b)
			}
		}
		d.cur.buf = d.cur.buf[n:]
		d.n += uint64(n)
		remaining -= n
		if len(d.cur.buf) == 0 {
			d.disposeBuffer()
		}
	}
	return nil
}

func (d *BufferedReader) DiscardUntil(b byte) error {
	for {
		err := d.buffer()
		if err != nil {
			return err
		}
		i := bytes.IndexByte(d.cur.buf, b)
		if i >= 0 {
			d.n += uint64(i)
			if annotate {
				for _, b := range d.cur.buf[:i] {
					d.samples.PushByte(b)
				}
			}
			d.cur.buf = d.cur.buf[i:]
			return nil
		}
		if annotate {
			for _, b := range d.cur.buf {
				d.samples.PushByte(b)
			}
		}
		d.n += uint64(len(d.cur.buf))
		d.disposeBuffer()
	}
}

type MVBStream struct {
	r *BufferedReader
	v bool
}

func NewMVBStream() *MVBStream {
	return &MVBStream{
		r: NewDoubleBufferedReader(os.Stdin),
	}
}

func (s *MVBStream) NextSample() (bool, error) {
	b, err := s.r.ReadByte()
	if err != nil {
		return false, err
	}
	s.v = b == signalHigh
	return s.v, nil
}

func (s *MVBStream) WaitUntilElapsed(samples int) (bool, error) {
	err := s.r.Discard(samples - 1)
	if err != nil {
		return false, err
	}
	return s.NextSample()
}

func (s *MVBStream) WaitUntilElapsedOrEdge(samples int, v1 bool) (bool, error) {
	if s.v != v1 {
		return s.v, nil
	}
	for i := 0; i < samples; i++ {
		v2, err := s.NextSample()
		if err != nil {
			return false, err
		}
		if v2 != v1 {
			break
		}
	}
	return s.v, nil
}

func (s *MVBStream) WaitUntilIdle(samples int) (bool, error) {
	for {
		v1 := s.v
		v2, err := s.WaitUntilElapsedOrEdge(samples, v1)
		if err != nil {
			return false, err
		}
		if v2 == v1 {
			return s.v, nil
		}
	}
}

func (s *MVBStream) WaitUntil(v bool) (bool, error) {
	b := signalLow
	if v {
		b = signalHigh
	}
	err := s.r.DiscardUntil(b)
	if err != nil {
		return false, err
	}
	return s.NextSample()
}

func (s *MVBStream) V() bool {
	return s.v
}

func (s *MVBStream) N() uint64 {
	return s.r.n
}

func (s *MVBStream) Annotate(text string) {
	if annotate {
		s.r.samples.Annotate(text)
	}
}

func (s *MVBStream) GetSamples() []Sample {
	if annotate {
		return s.r.samples.Get()
	}
	return nil
}

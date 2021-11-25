package mvb

import (
	"io"
	"os"
)

// samples per second
const SampleRate = 12_000_000

const (
	signalHigh = byte(0x02)
	signalLow  = byte(0x00)
)

const BufSize = SampleRate / 2

type BufferedReader struct {
	ready chan interface{}
	done  chan *buffer

	cur *buffer
	i   int

	n uint64
}

type buffer struct {
	arr [BufSize]byte
	buf []byte
}

func NewDoubleBufferedReader(r io.Reader) *BufferedReader {
	var bufs [2]buffer
	d := &BufferedReader{
		ready: make(chan interface{}, len(bufs)),
		done:  make(chan *buffer, len(bufs)),
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
	d.i = 0
}

func (d *BufferedReader) ReadByte() (byte, error) {
	err := d.buffer()
	if err != nil {
		return 0, err
	}
	b := d.cur.buf[d.i]
	d.i++
	d.n++
	if d.i >= len(d.cur.buf) {
		d.disposeBuffer()
	}
	return b, nil
}

func (d *BufferedReader) Discard(remaining int) error {
	for remaining > 0 {
		err := d.buffer()
		if err != nil {
			return err
		}
		available := len(d.cur.buf) - d.i
		if remaining <= available {
			d.i += remaining
			d.n += uint64(remaining)
			remaining = 0
		} else {
			d.i += available
			d.n += uint64(available)
			remaining -= available
		}
		if d.i >= len(d.cur.buf) {
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
		remaining := d.cur.buf[d.i:]
		for i, v := range remaining {
			if v == b {
				d.n += uint64(i)
				d.i += i
				return nil
			}
		}
		d.n += uint64(len(remaining))
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

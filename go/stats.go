package mvb

import "sort"

const (
	sparkSize    = 10
	errorLogSize = 10
	varLogSize   = 20
)

type Stats struct {
	Total     uint64
	rate      []uint64
	mrRates   [MR_AMOUNT][]uint64
	errorRate []uint64
	ErrorLog  []Error

	Vars map[uint16][]byte

	Capture *Capture
}

type Var struct {
	Port  uint16
	Value []byte
}

func NewStats() Stats {
	var mrRates [MR_AMOUNT][]uint64
	for i := range mrRates {
		mrRates[i] = newRate()
	}
	return Stats{
		rate:      newRate(),
		errorRate: newRate(),
		mrRates:   mrRates,
		ErrorLog:  make([]Error, 0, errorLogSize),
		Vars:      make(map[uint16][]byte),
	}
}

func newRate() []uint64 {
	return make([]uint64, sparkSize+1)
}

func rateView(a []uint64) []uint64 {
	return a[:len(a)-1]
}

func rateShift(a []uint64) {
	dst := a[:len(a)-1]
	src := a[1:]
	copy(dst, src)
	a[len(a)-1] = 0
}

func rateCount(a []uint64) {
	a[len(a)-1]++
}

func (s *Stats) Rate() []uint64 {
	return rateView(s.rate)
}

func (s *Stats) MRRate(mr MasterRequest) []uint64 {
	return rateView(s.mrRates[mr])
}

func (s *Stats) ErrorRate() []uint64 {
	return rateView(s.errorRate)
}

func (s *Stats) Tick() {
	rateShift(s.rate)
	rateShift(s.errorRate)
	for i := range s.mrRates {
		rateShift(s.mrRates[i])
	}
}

func (s *Stats) CountTelegram(t *Telegram) {
	s.Total++
	rateCount(s.rate)
	fcode := fcodes[t.Master.FCode]
	rateCount(s.mrRates[fcode.MasterRequest])
	if fcode.MasterRequest == MR_PROCESS_DATA && t.Slave != nil {
		s.SetVar(t.N(), t.Master.Address, t.Slave.data)
	}
}

func (s *Stats) SetVar(n uint64, port uint16, value []byte) {
	s.Vars[port] = value
	if s.Capture != nil && !s.Capture.Stopped {
		s.Capture.Add(n, port, value)
	}
}

func (s *Stats) CountError(err Error) {
	if len(s.ErrorLog) == cap(s.ErrorLog) {
		dst := s.ErrorLog[:len(s.ErrorLog)-1]
		src := s.ErrorLog[1:]
		copy(dst, src)
		s.ErrorLog = dst
	}
	s.ErrorLog = append(s.ErrorLog, err)
	rateCount(s.errorRate)
}

func (s *Stats) StartStopCapture() {
	if s.Capture != nil && !s.Capture.Stopped {
		s.Capture.Stopped = true
	} else {
		s.Capture = &Capture{
			Vars: make(map[uint16][]VarChange),
		}
	}
}

func (s *Stats) DiscardCapture() {
	s.Capture = nil
}

type Capture struct {
	Stopped   bool
	Vars      map[uint16][]VarChange
	SeenPorts []int
}

type VarChange struct {
	N     uint64
	Value []byte
}

func (c *Capture) Add(n uint64, port uint16, value []byte) {
	_, seen := c.Vars[port]
	if !seen {
		i := sort.SearchInts(c.SeenPorts, int(port))
		c.SeenPorts = append(c.SeenPorts, 0)
		copy(c.SeenPorts[i+1:], c.SeenPorts[i:])
		c.SeenPorts[i] = int(port)
	}

	c.Vars[port] = append(c.Vars[port], VarChange{N: n, Value: value})
}

package mvb

const (
	sparkSize    = 10
	errorLogSize = 10
)

type Stats struct {
	Total      uint64
	rate       []uint64
	fcodeRates [16][]uint64
	errorRate  []uint64
	ErrorLog   []Error
}

func NewStats() Stats {
	var fcodeRates [16][]uint64
	for i := range fcodeRates {
		fcodeRates[i] = newRate()
	}
	return Stats{
		rate:       newRate(),
		errorRate:  newRate(),
		fcodeRates: fcodeRates,
		ErrorLog:   make([]Error, 0, errorLogSize),
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

func (s *Stats) FCodeRate(fcode uint8) []uint64 {
	return rateView(s.fcodeRates[fcode])
}

func (s *Stats) ErrorRate() []uint64 {
	return rateView(s.errorRate)
}

func (s *Stats) Tick() {
	rateShift(s.rate)
	rateShift(s.errorRate)
	for i := range s.fcodeRates {
		rateShift(s.fcodeRates[i])
	}
}

func (s *Stats) CountTelegram(t *Telegram) {
	s.Total++
	rateCount(s.rate)
	rateCount(s.fcodeRates[t.Master.FCode])
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

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
	Errors     []Error
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
		Errors:     make([]Error, 0, errorLogSize),
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
	if len(s.Errors) == cap(s.Errors) {
		dst := s.Errors[:len(s.Errors)-1]
		src := s.Errors[1:]
		copy(dst, src)
		s.Errors = dst
	}
	s.Errors = append(s.Errors, err)
	rateCount(s.errorRate)
}

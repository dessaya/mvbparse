package mvb

type Stats struct {
	Total  uint64
	rate   []uint64
	Errors []Error
}

func NewStats() Stats {
	return Stats{
		rate:   make([]uint64, 61),
		Errors: make([]Error, 0, 10),
	}
}

func (s *Stats) Rate() []uint64 {
	return s.rate[:len(s.rate)-1]
}

func (s *Stats) Tick() {
	dst := s.rate[:len(s.rate)-1]
	src := s.rate[1:]
	copy(dst, src)
	s.rate[len(s.rate)-1] = 0
}

func (s *Stats) CountTelegram(t *Telegram) {
	s.Total++
	s.rate[len(s.rate)-1]++
}

func (s *Stats) CountError(err Error) {
	if len(s.Errors) == cap(s.Errors) {
		dst := s.Errors[:len(s.Errors)-1]
		src := s.Errors[1:]
		copy(dst, src)
		s.Errors = dst
	}
	s.Errors = append(s.Errors, err)
}

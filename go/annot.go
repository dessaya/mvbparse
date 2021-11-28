package mvb

import "strings"

type Samples struct {
	buffer []Sample
}

type Sample struct {
	V          byte
	Annotation string
}

func NewSamples(size int) *Samples {
	return &Samples{buffer: make([]Sample, size)}
}

func (b *Samples) PushByte(v byte) {
	a := b.buffer
	dst := a[:len(a)-1]
	src := a[1:]
	copy(dst, src)
	a[len(a)-1] = Sample{v, ""}
}

func (b *Samples) Annotate(s string) {
	a := b.buffer
	a[len(a)-1].Annotation = s
}

func (b *Samples) Get() []Sample {
	s := make([]Sample, len(b.buffer))
	copy(s, b.buffer)
	return s
}

func traceSamples(ss []Sample) string {
	var sb strings.Builder
	sb.WriteString("[")
	for _, s := range ss {
		switch s.V {
		case signalHigh:
			sb.WriteString("+")
		case signalLow:
			sb.WriteString(".")
		default:
			panic("unreachable")
		}
		if s.Annotation != "" {
			sb.WriteString("[")
			sb.WriteString(s.Annotation)
			sb.WriteString("]")
		}
	}
	sb.WriteString("]")
	return sb.String()
}

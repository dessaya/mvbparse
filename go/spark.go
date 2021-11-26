package mvb

import (
	"strings"
)

func spark(vs []uint64) string {
	min := vs[0]
	max := vs[0]
	for _, v := range vs {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if min == max {
		return strings.Repeat("▁", len(vs))
	}
	rs := make([]rune, len(vs))
	f := float64(8) / float64(max-min)
	for j, v := range vs {
		i := rune(f * float64(v-min))
		if i > 7 {
			i = 7
		}
		rs[j] = '▁' + i
	}
	return string(rs)
}

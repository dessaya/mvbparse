package mvb

import "flag"

func InitFlags() {
	initInputFlags()
	initDashboardFlags()
	flag.Parse()
}

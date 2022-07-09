package mvb

import "flag"

var VerboseFlag bool

func InitFlags() {
	initInputFlags()
	initDashboardFlags()
	flag.BoolVar(&VerboseFlag, "v", false, "verbose")
	flag.Parse()
}

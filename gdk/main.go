package main

import (
	"flag"
	"fmt"
	"os"

	env "github.com/ClarkGuan/go-android-env"
)

func main() {
	apiLevel := 21
	flag.IntVar(&apiLevel, "level", 21, "android API level")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "缺少参数，gdk 不能自己运行\n")
		os.Exit(1)
	}

	env.LibsMain(apiLevel, flag.Args()...)
}

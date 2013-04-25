package main

import (
	"flag"
	"log"
	"pacificauploaderd/auto"
	"pacificauploaderd/common"
	"pacificauploaderd/upload"
	"pacificauploaderd/web"
	"os"
)

var quit chan bool

var disableAuto bool

func main() {
	log.Println("Entering pacificauploaderd main")
	processArgs()
	common.Init()
	web.Init()
	bm := upload.Init()
	if disableAuto == false {
		auto.Init(bm)
	}
	web.ServerRun()
	<-quit
}

func usage() {
	log.Printf("usage: goclient [-basedir]")
	flag.PrintDefaults()
	os.Exit(1)
}

func processArgs() {
	flag.BoolVar(&common.Profiler, "profiler", false, "Enable the profiler.")
	flag.BoolVar(&common.Devel, "devel", false, "Run out of source tree.")
	flag.BoolVar(&common.System, "system", false, "Run in system mode.")
	flag.BoolVar(&disableAuto, "disable-auto", false, "Disable the auto uploader.")
	flag.StringVar(&common.BaseDir, "basedir", common.DefaultBaseDirGet(), "Set the base directory.")
	flag.Usage = usage
	flag.Parse()
}

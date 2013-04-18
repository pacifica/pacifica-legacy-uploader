package main

import (
	"flag"
	"fmt"
	"net/rpc"
	"os"
	"pacifica/pipepair"
)

type Server struct{}

var (
	user string
)

func main() {
	processArgs()
	pipepair := pipepair.PipePair{In: os.Stdin, Out: os.Stdout}
	server := Server{}
	rpc.Register(server)
	rpc.DefaultServer.ServeConn(pipepair)
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: userd")
	flag.PrintDefaults()
	os.Exit(1)
}

func processArgs() {
	flag.Usage = usage
	flag.Parse()
}

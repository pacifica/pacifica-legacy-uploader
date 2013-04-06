package main

import (
	"fmt"
	"flag"
	"os/exec"
	"net/rpc"
	"pacifica/pipepair"
	userdrpc "pacificauploaderuserd/rpc"
)

func main() {
	flag.Parse()
	args := flag.Args()
	cmd := exec.Command("./pacificauploaderuserd")
	pipe_write, _ := cmd.StdinPipe()
	pipe_read, _ := cmd.StdoutPipe()
	pipepair := pipepair.PipePair{In: pipe_read, Out: pipe_write}
	_ = cmd.Start()
	client := rpc.NewClient(pipepair)
	for i := 0; i < len(args); i++ {
		var reply bool;
		if err := client.Call(userdrpc.ACCESS, userdrpc.AccessArgs{Path: args[i]}, &reply); err != nil {
			fmt.Printf("%v\n", err)
		} else {
			fmt.Printf("Access %s = %t\n", args[i], reply)
		}
	}
}

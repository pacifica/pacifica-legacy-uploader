package main

import (
	"syscall"
	userdrpc "pacificauploaderuserd/rpc"
)

func (s Server) Access(args *userdrpc.AccessArgs, reply *bool) error {
//FIXME No constants please.
	*reply = (syscall.Access(args.Path, 4)) == nil
	return nil
}

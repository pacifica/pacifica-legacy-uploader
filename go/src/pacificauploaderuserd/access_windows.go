package main

import (
	userdrpc "pacificauploaderuserd/rpc"
)

func (s Server) Access(args *userdrpc.AccessArgs, reply *bool) error {
//FIXME Make this work
	*reply = true
	return nil
}

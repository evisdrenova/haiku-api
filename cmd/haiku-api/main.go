package main

import (
	"flag"
	"fmt"
	"net"

	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

func main() {
	logger := newLogger()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		logger.Error(err, "failed to listen")
		return
	}

	srvr := grpc.NewServer()
	err = registerServices(srvr, logger)
	if err != nil {
		logger.Error(err, "failed to listen")
		return
	}

	logger.Info("server listening", "address", lis.Addr().String())
	err = srvr.Serve(lis)
	if err != nil {
		logger.Error(err, "failed to serve")
		return
	}
}

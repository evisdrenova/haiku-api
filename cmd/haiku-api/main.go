package main

import (
	"flag"
	"fmt"
	"net"
)

var (
	port = flag.Int("port", 50051, "The server port")
	// flip the default to this in order to connect to the cluster
	// /Users/marco/.kube/config
	kubeConfigPath = flag.String("kube-config-path", "", "(optional) the path to the kube config file to be used")
)

func main() {
	logger := newLogger()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		logger.Error(err, "failed to listen")
		return
	}

	srvr, err := registerServices(*kubeConfigPath, logger)
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

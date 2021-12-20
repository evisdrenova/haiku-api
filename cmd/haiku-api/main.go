package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
)

var (
	portFlag = flag.Int("port", 50051, "The server port")
	// flip the default to this in order to connect to the cluster
	// /Users/marco/.kube/config
	kubeConfigPath = flag.String("kube-config-path", "", "(optional) the path to the kube config file to be used")
)

func main() {
	flag.Parse()
	logger := newLogger()
	port, err := getPort()
	if err != nil {
		logger.Error(err, "couldn't parse port")
	}

	logger.Info(fmt.Sprintf("listening on port %d", port))
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		logger.Error(err, "failed to listen")
		return
	}

	logger.Info(fmt.Sprintf("kube.config: %s", *kubeConfigPath))
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

func getPort() (int, error) {
	strPort := os.Getenv("PORT")
	if strPort != "" {
		port, err := strconv.Atoi(strPort)
		if err != nil {
			return 0, err
		}
		return port, nil
	}

	if portFlag == nil {
		return 0, fmt.Errorf("port flag unset")
	}
	return *portFlag, nil
}

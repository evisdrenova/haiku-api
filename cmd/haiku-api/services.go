package main

import (
	"github.com/go-logr/logr"
	v1 "github.com/mhelmich/haiku-api/pkg/api/v1"
	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"google.golang.org/grpc"
)

func registerServices(srvr *grpc.Server, logger logr.Logger) error {
	cliSrvr, err := v1.NewCliServer("kube.config", logger)
	if err != nil {
		return err
	}

	pb.RegisterCliServiceServer(srvr, cliSrvr)
	return nil
}

package main

import (
	"github.com/go-logr/logr"
	"github.com/mhelmich/haiku-api/pkg/api"
	"github.com/mhelmich/haiku-api/pkg/api/pb"
	"google.golang.org/grpc"
)

func registerServices(srvr *grpc.Server, logger logr.Logger) error {
	cliSrvr, err := api.NewCliServer("kube.config", logger)
	if err != nil {
		return err
	}

	pb.RegisterCliServiceServer(srvr, cliSrvr)
	return nil
}

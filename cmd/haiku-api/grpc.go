package main

import (
	"github.com/go-logr/logr"
	"github.com/mercari/go-grpc-interceptor/xrequestid"
	v1 "github.com/mhelmich/haiku-api/pkg/api/v1"
	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"google.golang.org/grpc"
)

func registerServices(logger logr.Logger) (*grpc.Server, error) {
	srvr := newGrpcServer()
	cliSrvr, err := v1.NewCliServer("kube.config", logger)
	if err != nil {
		return nil, err
	}

	pb.RegisterCliServiceServer(srvr, cliSrvr)
	return srvr, nil
}

func newGrpcServer() *grpc.Server {
	return grpc.NewServer(
		grpc.UnaryInterceptor(xrequestid.UnaryServerInterceptor()),
		grpc.StreamInterceptor(xrequestid.StreamServerInterceptor()),
	)
}

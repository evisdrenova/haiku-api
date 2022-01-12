package main

import (
	"github.com/go-logr/logr"
	v1 "github.com/mhelmich/haiku-api/pkg/api/v1"
	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"github.com/mhelmich/haiku-api/pkg/requestid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func registerServices(configPath string, logger logr.Logger) (*grpc.Server, error) {
	srvr, err := newGrpcServer()
	if err != nil {
		return nil, err
	}

	cliSrvr, err := v1.NewCliServer(configPath, logger)
	if err != nil {
		return nil, err
	}

	pb.RegisterCliServiceServer(srvr, cliSrvr)
	return srvr, nil
}

func newGrpcServer() (*grpc.Server, error) {
	creds, err := credentials.NewServerTLSFromFile("keys/service.pem", "keys/service.key")
	if err != nil {
		return nil, err
	}

	return grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(requestid.UnaryServerInterceptor()),
		grpc.StreamInterceptor(requestid.StreamServerInterceptor()),
	), nil
}

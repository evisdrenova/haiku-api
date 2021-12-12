package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/mhelmich/haiku-api/pkg/api"
	"github.com/mhelmich/haiku-api/pkg/api/pb"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

func registerServices(srvr *grpc.Server, logger logr.Logger) error {
	cliSrvr, err := api.NewCliServer("kube.config", logger)
	if err != nil {
		return err
	}

	pb.RegisterCliServiceServer(srvr, cliSrvr)
	return nil
}

func newLogger() logr.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerologr.NameFieldName = "logger"
	zerologr.NameSeparator = "/"

	zl := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	zl = zl.With().Caller().Timestamp().Logger()
	return zerologr.New(&zl)
}

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

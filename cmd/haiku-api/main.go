package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/mhelmich/haiku-api/pkg/api"
	"github.com/mhelmich/haiku-api/pkg/api/pb"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srvr := grpc.NewServer()
	pb.RegisterCliServiceServer(srvr, &api.CliServer{})
	log.Printf("server listening at %v", lis.Addr())
	err = srvr.Serve(lis)
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

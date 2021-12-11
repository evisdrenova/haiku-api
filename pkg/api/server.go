package api

import (
	"context"

	"github.com/mhelmich/haiku-api/pkg/api/pb"
)

type CliServer struct {
	pb.UnimplementedCliServiceServer
}

// This will have to create a k8s namespace and likely more stuff.
func (s *CliServer) Init(ctx context.Context, req *pb.InitRequest) (*pb.InitReply, error) {
	return nil, nil
}

// This will have to update a k8s manifest with a new version number of a container image.
func (s *CliServer) Deploy(ctx context.Context, req *pb.DeployRequest) (*pb.DeployReply, error) {
	return nil, nil
}

// The env family of endpoints maybe gets stored as a k8s configmap. Who knows...
func (s *CliServer) ListEnv(ctx context.Context, req *pb.ListEnvRequest) (*pb.ListEnvReply, error) {
	return nil, nil
}

func (s *CliServer) SetEnv(ctx context.Context, req *pb.SetEnvRequest) (*pb.SetEnvReply, error) {
	return nil, nil
}

func (s *CliServer) RemoveEnv(ctx context.Context, req *pb.RemoveEnvRequest) (*pb.RemoveEnvReply, error) {
	return nil, nil
}

// This will have to create a k8s secret (and maybe patch that secret to the local service account).
func (s *CliServer) DockerLogin(ctx context.Context, req *pb.DockerLoginRequest) (*pb.DockerLoginReply, error) {
	return nil, nil
}

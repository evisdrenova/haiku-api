package api

import (
	"context"
	"io/ioutil"

	"github.com/go-logr/logr"
	"github.com/mhelmich/haiku-api/pkg/api/pb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func KubeConfigGetter(path string) clientcmd.KubeconfigGetter {
	return func() (*clientcmdapi.Config, error) {
		bites, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return clientcmd.Load(bites)
	}
}

func NewCliServer(configPath string, logger logr.Logger) (*CliServer, error) {
	config, err := clientcmd.BuildConfigFromKubeconfigGetter("", KubeConfigGetter(configPath))
	if err != nil {
		return nil, err
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &CliServer{
		k8sClient: k8sClient,
	}, nil
}

type CliServer struct {
	pb.UnimplementedCliServiceServer

	k8sClient *kubernetes.Clientset
	logger    logr.Logger
}

// This will have to create a k8s namespace and likely more stuff.
func (s *CliServer) Init(ctx context.Context, req *pb.InitRequest) (*pb.InitReply, error) {
	s.logger.Info("init namespace %s", req.ProjectName)
	k8sNamespace, err := s.k8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.ProjectName,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, ErrAlreadyExists
		}

		return nil, err
	}

	return &pb.InitReply{
		ID: string(k8sNamespace.UID),
	}, nil
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

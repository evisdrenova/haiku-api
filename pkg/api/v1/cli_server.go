package v1

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"github.com/mhelmich/haiku-api/pkg/requestid"
	ho "github.com/mhelmich/haiku-operator/apis/entities/v1alpha1"
	"github.com/mhelmich/haiku-operator/apis/serving/v1alpha1"
	hc "github.com/mhelmich/haiku-operator/clientset"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

// TODO: use fancy option pattern instead of this hack
func NewCliServer(configPath string, logger logr.Logger) (*CliServer, error) {
	var config *rest.Config
	var err error
	if configPath == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromKubeconfigGetter("", KubeConfigGetter(configPath))
	}
	if err != nil {
		return nil, err
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	haikuClient, err := hc.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &CliServer{
		k8sClient:   k8sClient,
		haikuClient: haikuClient,
		logger:      logger,
	}, nil
}

type CliServer struct {
	pb.UnimplementedCliServiceServer

	k8sClient   *kubernetes.Clientset
	haikuClient *hc.Clientset
	logger      logr.Logger
}

// This will have to create a k8s namespace and likely more stuff.
func (s *CliServer) Init(ctx context.Context, req *pb.InitRequest) (*pb.InitReply, error) {
	logger := s.logger.WithValues("namespaceName", req.EnvironmentName, "requestID", requestid.FromContext(ctx))
	logger.Info("init namespace")
	k8sNamespace, err := s.k8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.EnvironmentName,
		},
	}, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		logger.Info("environment already exists")
		return nil, ErrAlreadyExists
	} else if err != nil {
		logger.Error(err, "failed to create environment")
		return nil, err
	}

	return &pb.InitReply{
		ID: string(k8sNamespace.UID),
	}, nil
}

// This will have to create or update a haiku service manifest.
// Those are relatively simple and be pulled in from the haiku operator.
// The main attribute of those is the image url.
func (s *CliServer) Deploy(ctx context.Context, req *pb.DeployRequest) (*pb.DeployReply, error) {
	logger := s.logger.WithValues("namespaceName", req.EnvironmentName, "requestID", requestid.FromContext(ctx))
	logger.Info("deploy namespace")
	service, err := s.haikuClient.ServingV1alpha1().Services(req.EnvironmentName).Create(ctx, &v1alpha1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.EnvironmentName,
			Name:      req.ServiceName,
		},
		Spec: v1alpha1.ServiceSpec{
			Image: req.Image,
		},
	}, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		logger.Info("service already exists")
		return nil, ErrAlreadyExists
	} else if err != nil {
		logger.Error(err, "failed to create service")
		return nil, err
	}

	watcher, err := s.haikuClient.ServingV1alpha1().Services(req.EnvironmentName).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "failed to create watcher for service")
		return nil, err
	}

	serviceURL, err := waitForServiceURLUpdated(ctx, watcher, logger)
	if err != nil {
		logger.Error(err, "failed to watch service")
		return nil, err
	}

	return &pb.DeployReply{
		URL: serviceURL,
		ID:  string(service.UID),
	}, nil
}

func waitForServiceURLUpdated(ctx context.Context, watcher watch.Interface, logger logr.Logger) (string, error) {
	// it's safe be called multiple times
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			// request timed out
			return "", fmt.Errorf("timeout")
		case event := <-watcher.ResultChan():
			svc, ok := event.Object.(*v1alpha1.Service)
			if !ok {
				logger.Error(fmt.Errorf("object was %T", event.Object), "couldn't cast event watcher object to service")
			}
			if svc.Status.URL != "" {
				return svc.Status.URL, nil
			}
		}
	}
}

// The env family of endpoints maybe gets stored as a k8s configmap.
// The config map needs to be created in a particular namespace and with particular labels.
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
// As illustrated here: https://knative.dev/docs/serving/deploying-from-private-registry/
func (s *CliServer) DockerLogin(ctx context.Context, req *pb.DockerLoginRequest) (*pb.DockerLoginReply, error) {
	// TODO: write a "getK8sNamespaceForHaikuSpaceName" function
	namespaceName := req.EnvironmentName
	secretName := fmt.Sprintf("docker-%s-%s", uuid.NewString(), req.Server)
	logger := s.logger.WithValues("namespaceName", namespaceName, "requestID", requestid.FromContext(ctx))
	logger.Info("creating dockerlogin")
	dl := &ho.DockerLogin{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespaceName,
			Name:      secretName,
		},
		Spec: ho.DockerLoginSpec{
			Server:   req.Server,
			Username: req.Username,
			Password: req.Password,
			Email:    req.Email,
		},
	}
	dl, err := s.haikuClient.EntitiesV1alpha1().DockerLogins(namespaceName).Create(ctx, dl, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		logger.Info("dockerlogin already exists")
		return nil, ErrAlreadyExists
	} else if err != nil {
		logger.Error(err, "failed to create dockerlogin")
		return nil, err
	}

	return &pb.DockerLoginReply{
		ID: string(dl.UID),
	}, nil
}

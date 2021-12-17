package v1

import (
	"context"
	"io/ioutil"

	"github.com/go-logr/logr"
	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"github.com/mhelmich/haiku-api/pkg/requestid"
	ho "github.com/mhelmich/haiku-operator/apis/entities/v1alpha1"
	"github.com/mhelmich/haiku-operator/apis/serving/v1alpha1"
	hc "github.com/mhelmich/haiku-operator/clientset"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	logger := s.logger.WithValues("namespaceName", req.ProjectName, "requestID", requestid.FromContext(ctx))
	logger.Info("init namespace")
	k8sNamespace, err := s.k8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.ProjectName,
		},
	}, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		return nil, ErrAlreadyExists
	} else if err != nil {
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
	requestID := requestid.FromContext(ctx)
	s.logger.Info("deploy namespace", "namespaceName", req.ProjectName, "requestID", requestID)
	// _, err := s.k8sClient.CoreV1().Namespaces().Get(ctx, req.ProjectName, metav1.GetOptions{})
	// if err != nil {
	// 	// throw error saying projectName does not exist
	// 	return nil, err
	// }
	// Somehow deploy to knative deployment to provided namespace
	// If it already exists, we should update the deployment to include the new docker image/tag (if it's not current)
	// https://knative.dev/docs/reference/api/serving-api/#serving.knative.dev%2fv1
	// knService, err := s.k8sClient.CoreV1().Services(req.ProjectName).Apply(ctx, &v1.ServiceApplyConfiguration{}, metav1.ApplyOptions{})

	// s.haikuClient.EntitiesV1alpha1().
	service, err := s.haikuClient.ServingV1alpha1().Services(req.ProjectName).Create(ctx, &v1alpha1.Service{
		Spec: v1alpha1.ServiceSpec{
			Image: req.Image,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: req.ServiceName,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		return nil, err
	}

	return &pb.DeployReply{
		URL: "", // how do we get the URL? :D
		ID:  string(service.UID),
	}, err
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
	namespaceName := "test-api"
	logger := s.logger.WithValues("namespaceName", namespaceName, "requestID", requestid.FromContext(ctx))
	logger.Info("creating docker login")
	dl := &ho.DockerLogin{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespaceName,
			Name:      "super-secret",
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
		return nil, ErrAlreadyExists
	} else if err != nil {
		return nil, err
	}

	return &pb.DockerLoginReply{
		ID: string(dl.UID),
	}, nil
}

package v1

import (
	"context"
	"io/ioutil"

	"github.com/go-logr/logr"
	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"github.com/mhelmich/haiku-api/pkg/requestid"
	ho "github.com/mhelmich/haiku-operator/apis/entities/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
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

func newRestClient(config *rest.Config) (*rest.RESTClient, error) {
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &ho.GroupVersion
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()
	return rest.UnversionedRESTClientFor(&crdConfig)
}

// TODO: use fancy option pattern instead of this hack
func NewCliServer(configPath string, logger logr.Logger) (*CliServer, error) {
	var config *rest.Config
	var err error
	// register our CRDs with the client
	ho.AddToScheme(scheme.Scheme)

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

	unversionedRestClient, err := newRestClient(config)
	if err != nil {
		return nil, err
	}

	return &CliServer{
		k8sClient:             k8sClient,
		unversionedRestClient: unversionedRestClient,
		logger:                logger,
	}, nil
}

type CliServer struct {
	pb.UnimplementedCliServiceServer

	k8sClient             *kubernetes.Clientset
	unversionedRestClient *rest.RESTClient
	logger                logr.Logger
}

// This will have to create a k8s namespace and likely more stuff.
func (s *CliServer) Init(ctx context.Context, req *pb.InitRequest) (*pb.InitReply, error) {
	requestID := requestid.FromContext(ctx)
	s.logger.Info("init namespace", "namespaceName", req.ProjectName, "requestID", requestID)
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

// This will have to create or update a haiku service manifest.
// Those are relatively simple and be pulled in from the haiku operator.
// The main attribute of those is the image url.
func (s *CliServer) Deploy(ctx context.Context, req *pb.DeployRequest) (*pb.DeployReply, error) {
	return nil, nil
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
	dl := &ho.DockerLogin{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-api",
			Name:      "super-secret",
		},
		Spec: ho.DockerLoginSpec{
			Server:   req.Server,
			Username: req.Username,
			Password: req.Password,
			Email:    req.Email,
		},
	}
	created := &ho.DockerLogin{}
	err := s.unversionedRestClient.Post().
		Resource("dockerlogins").
		Namespace("test-api").
		Body(dl).
		Do(ctx).
		Into(created)
	if err != nil {
		return nil, err
	}

	return &pb.DockerLoginReply{
		ID: string(created.UID),
	}, nil
}

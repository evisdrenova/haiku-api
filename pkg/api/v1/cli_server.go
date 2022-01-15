package v1

import (
	"context"
	goerrors "errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	storage "cloud.google.com/go/storage"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"github.com/mhelmich/haiku-api/pkg/requestid"
	hc "github.com/mhelmich/haiku-operator/clientset"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonclients "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"google.golang.org/api/option"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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

	gcsClient, err := getGcsClient(context.Background())
	if err != nil {
		return nil, err
	}

	tektonClient, err := tektonclients.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &CliServer{
		k8sClient:    k8sClient,
		haikuClient:  haikuClient,
		gcsClient:    gcsClient,
		tektonClient: tektonClient,
		logger:       logger,
	}, nil
}

type CliServer struct {
	pb.UnimplementedCliServiceServer

	k8sClient    *kubernetes.Clientset
	haikuClient  *hc.Clientset
	gcsClient    *storage.Client
	tektonClient *tektonclients.Clientset
	logger       logr.Logger
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

func (s *CliServer) GetServiceUploadUrl(ctx context.Context, req *pb.GetServiceUploadUrlRequest) (*pb.GetServiceUploadUrlResponse, error) {
	bucket := s.gcsClient.Bucket("haiku_service_storage") // todo: make this an environment variable

	if bucket == nil {
		return nil, goerrors.New("unable to find bucket")
	}

	uploadKey := getUrlUploadKey(req.EnvironmentName, req.ServiceName)
	signedURL, err := bucket.SignedURL(uploadKey, &storage.SignedURLOptions{
		Scheme:      storage.SigningSchemeV4,
		Method:      "PUT",
		ContentType: "application/zip",
		Expires:     time.Now().Add(15 * time.Minute),
	})
	if err != nil {
		return nil, err
	}

	return &pb.GetServiceUploadUrlResponse{
		URL:       signedURL,
		UploadKey: fmt.Sprintf("gs://%s/%s", "haiku_service_storage", uploadKey),
	}, nil
}

func (s *CliServer) DeployUrl(req *pb.DeployUrlRequest, stream pb.CliService_DeployUrlServer) error {
	ctx := stream.Context()
	logger := s.logger.WithValues("requestID", requestid.FromContext(ctx))
	// TODO(marco): user input sanitation
	// think base58 encoding or so
	imageURI := fmt.Sprintf("harbor.haiku.icu/library/%s-%s", req.EnvironmentName, req.ServiceName)
	prName, err := triggerPipeline(ctx, s.tektonClient, req.EnvironmentName, req.ServiceName, req.URL, imageURI)
	if err != nil {
		return err
	}

	logger.Info("triggered pipeline successfully: %s", prName)
	watcher, err := s.tektonClient.TektonV1beta1().PipelineRuns("haiku-runtimes").Watch(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, prName).String(),
	})
	if err != nil {
		logger.Error(err, "failed to create watcher for service")
		return err
	}

	logger.Info("watching pipeline run...")
	return watchPipelineRun(ctx, watcher, stream, prName, logger)
}

func watchPipelineRun(ctx context.Context, watcher watch.Interface, stream pb.CliService_DeployUrlServer, prName string, logger logr.Logger) error {
	defer watcher.Stop()
	var msg string
	var taskIdx int

	for {
		select {
		case <-ctx.Done():
			// request timed out
			return fmt.Errorf("request timeout")
		case <-time.After(5 * time.Minute):
			return fmt.Errorf("server timeout")
		case event := <-watcher.ResultChan():
			pr, ok := event.Object.(*tekton.PipelineRun)
			if !ok {
				logger.Error(fmt.Errorf("object was %T", event.Object), "couldn't cast event watcher object to pipelinerun")
			}

			msg, taskIdx = composeClientMessage(pr.Status, taskIdx)
			err := stream.Send(&pb.DeployUrlReply{
				Data: &pb.DeployUrlReply_DeploymentUpdate{
					DeploymentUpdate: &pb.DeploymentUpdate{
						Message: msg,
					},
				},
			})
			if err != nil {
				return err
			}

			// status jumps from "unknown" to either "true" or "false" when the PR finished
			if len(pr.Status.Conditions) > 0 && pr.Status.Conditions[0].Status != corev1.ConditionUnknown {
				var url string
				for _, res := range pr.Status.PipelineResults {
					if res.Name == "url" {
						url = res.Value
						break
					}
				}
				err = stream.Send(&pb.DeployUrlReply{
					Data: &pb.DeployUrlReply_URL{
						URL: url,
					},
				})
				if err != nil {
					return err
				}

				return nil
			}
		}
	}
}

func composeClientMessage(prStatus tekton.PipelineRunStatus, taskIdx int) (string, int) {
	if len(prStatus.Conditions) == 0 {
		return "waiting for your deployment to start", 0
	}

	switch len(prStatus.TaskRuns) {
	case 1:
		if taskIdx == 1 {
			return ".", 1
		}

		return "finding your code", 1
	case 2:
		if taskIdx == 2 {
			return ".", 2
		}

		return "injecting execution context", 2
	case 3:
		if taskIdx == 3 {
			return ".", 3
		}

		return "baking", 3
	case 4:
		if taskIdx == 4 {
			return ".", 4
		}

		return "finding a home", 4
	default:
		return "", -1
	}
}

func getGcsClient(ctx context.Context) (*storage.Client, error) {
	// The credentials can be ommitted as GOOGLE_APPLICATION_CREDENTIALS is the default, but I think it's better to be clear
	// about how we are loading in the credentials
	// return storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))

	return storage.NewClient(ctx, option.WithCredentialsFile("/Users/evisdrenova/Documents/code/Haiku/haiku-cli/lofty-tea-334923-ad7995b148b9.json"))
}

// https://cloud.google.com/storage/docs/naming-objects
var UPLOAD_KEY_REPLACER *strings.Replacer = strings.NewReplacer("/", "", "#", "", "[", "", "]", "", "?", "", "*", "")

func getUrlUploadKey(environmentName string, serviceName string) string {
	sanitizedEnvironmentName := UPLOAD_KEY_REPLACER.Replace(environmentName)
	sanitizedServiceName := UPLOAD_KEY_REPLACER.Replace(serviceName)
	timestampUnix := strconv.FormatInt(time.Now().Unix(), 10)
	return sanitizedEnvironmentName + "/" + sanitizedServiceName + "/" + timestampUnix + "_" + uuid.New().String() + ".zip"
}

func triggerPipeline(ctx context.Context, tektonClient *tektonclients.Clientset, envName string, serviceName string, gcsURI string, imageURL string) (string, error) {
	log.Printf("triggering pipeline...")
	pvFs := corev1.PersistentVolumeFilesystem
	twoHundertFiffyMeg, err := resource.ParseQuantity("250Mi")
	if err != nil {
		return "", err
	}

	pr := &tekton.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    "haiku-runtimes",
			GenerateName: "fastapi-pipeline-",
		},
		Spec: tekton.PipelineRunSpec{
			ServiceAccountName: "build-bot",
			PipelineRef: &tekton.PipelineRef{
				Name: "fastapi-pipeline",
			},
			Params: []tekton.Param{
				{
					Name:  "env-name",
					Value: *tekton.NewArrayOrString(envName),
				},
				{
					Name:  "service-name",
					Value: *tekton.NewArrayOrString(serviceName),
				},
				{
					Name:  "gcs-uri",
					Value: *tekton.NewArrayOrString(gcsURI),
				},
				{
					Name:  "image-url",
					Value: *tekton.NewArrayOrString(imageURL),
				},
				{
					Name:  "repo-url",
					Value: *tekton.NewArrayOrString("https://github.com/mhelmich/fastapi-starter"),
				},
			},
			Workspaces: []tekton.WorkspaceBinding{
				{
					Name: "workdir",
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							VolumeMode:  &pvFs,
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceStorage: twoHundertFiffyMeg,
								},
							},
						},
					},
				},
				{
					Name: "docker-registry-secrets",
					Secret: &corev1.SecretVolumeSource{
						SecretName: "docker-registry-harbor-core",
						Items: []corev1.KeyToPath{
							{
								Key:  "tls.crt",
								Path: "ca.crt",
							},
						},
					},
				},
				{
					Name: "gcs-credentials",
					Secret: &corev1.SecretVolumeSource{
						SecretName: "gcs-secret",
					},
				},
			},
		},
	}
	pr, err = tektonClient.TektonV1beta1().PipelineRuns("haiku-runtimes").Create(ctx, pr, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	fmt.Printf("created pipeline run: %s\n", pr.Name)
	return pr.Name, nil
}

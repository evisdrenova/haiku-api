#!/bin/bash

minikube start --cpus 5 --memory 10g --disk-size 40g
kubectl config view > kube.config

# install knative serving
# https://knative.dev/docs/install/serving/install-serving-with-yaml/
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.0.0/serving-crds.yaml
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.0.0/serving-core.yaml
kubectl wait --for=condition=Ready pods --all -n knative-serving

# install kourier
kubectl apply -f https://github.com/knative/net-kourier/releases/download/knative-v1.0.0/kourier.yaml
kubectl patch configmap/config-network \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'

# install magic domains
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.0.0/serving-default-domain.yaml

# install autoscaling
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.0.0/serving-hpa.yaml

echo "\n==================================="
echo "your minikube ip is: $(minikube ip)"
echo "===================================\n"

# tunnel asked you for you root passwd because I tries to bind ports 80 and 443
# after you entered your passwd, it just seems as it hangs. but tunnel just tunnels
# and never prints anything
minikube tunnel

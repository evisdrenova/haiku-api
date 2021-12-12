# haiku-api

## Setup a minikube backend

* `brew install minikube make docker`
* `make minikube`

The make target creates and starts a minikube cluster (assuming you have docker running). After some time, it asks for your sudo password. It does that because it wants to bind port 80 and 443. After that it just seems to hang. But in reality it established a tunnel and proxies traffic from your host to minikube.

After you run that target, you will also find a `kube.config` file in our repo root. That file serves as input to `haiku-api`. Right  now the file name and path are hard-coded.

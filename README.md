# haiku-api

## Setup a minikube backend

* `brew install minikube make docker`
* `make minikube`

The make target creates and starts a minikube cluster (assuming you have docker running). After some time, it asks for your sudo password. It does that because it wants to bind port 80 and 443. After that it just seems to hang. But in reality it established a tunnel and proxies traffic from your host to minikube.

After you run that target, you will also find a `kube.config` file in our repo root. That file serves as input to `haiku-api`. Right now the file name and path are hard-coded.

## Google Service Account Setup
A service account is required for some operations in the API.
Navigate to the [haiku-api service account](https://console.cloud.google.com/iam-admin/serviceaccounts/details/114241824999079558656/keys?project=lofty-tea-334923) and create a key. Be sure to download in JSON format.

Drop the file in the `keys` folder and name it `secret-google-service-account.json` Note: this is done purely for convenience as it has been git ignored, but as long as the prefix is `secret-*.json`
Next, set the environment variable in your shell prior to startup:

```sh
export GOOGLE_APPLICATION_CREDENTIALS=<path-to-keyfile>
```

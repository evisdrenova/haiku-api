# Build the manager binary
FROM golang:1.17 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# make it so that the local ssh key is mounted into the container
# in order to get that done, we need:
# * switch from https to ssh auth
# * install the openssh-client
# * use it to add github as known_hosts (otherwise the it will ask you to verify the ssh key fingerprint and fail)
# * use the mounted ssh key to download the dependencies
# RUN git config --global url."git@github.com:".insteadOf "https://github.com/"
# RUN apt-get install -y openssh-client
# RUN mkdir /root/.ssh/ && touch /root/.ssh/known_hosts && ssh-keyscan github.com > /root/.ssh/known_hosts
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
# RUN --mount=type=ssh go mod download

# Copy the go source
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o haiku-api cmd/haiku-api/*.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/haiku-api .
COPY keys/ keys/
# COPY kube.config .
USER 65532:65532

ENTRYPOINT ["/haiku-api"]

GO=go
GIT=git
PROTOC=protoc
DOCKER=docker

GIT_COMMIT ?= $(shell git rev-parse --short HEAD || echo unknown)
IMAGE_NAME = ghcr.io/mhelmich/haiku-api:v0.1.0-$(GIT_COMMIT)

default: all

all: build test
.PHONY: all

vendor:
	$(GO) mod tidy && $(GO) mod vendor
.PHONY: vendor

build: protos
	$(GO) build -o haiku-api cmd/haiku-api/*.go
.PHONY: build

test:
	$(GO) test ./... -race -v
.PHONY: test

fuckit:
	$(GO) clean --modcache && $(GIT) reset --hard HEAD && $(GIT) clean -fdx
.PHONY: fuckit

minikube:
	/bin/sh ./hack/setup_minikube.sh
.PHONY: minikube

protos:
	$(PROTOC) -I./protos/v1 --go_out=./pkg/api/v1/pb --go_opt=paths=source_relative --go-grpc_out=./pkg/api/v1/pb --go-grpc_opt=paths=source_relative protos/v1/cli.proto
.PHONY: protos

run: build
	./haiku-api
.PHONY: run

docker-build:
	$(DOCKER) build . -t $(IMAGE_NAME)
.PHONY: docker-build

docker-push:
	$(DOCKER) push $(IMAGE_NAME)
.PHONY: docker-push

certs:
	/bin/sh ./hack/create_certs.sh
.PHONY: certs

print-%  : ; @echo $* = $($*)

help:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
.PHONY: help

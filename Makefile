GO=go
GIT=git
PROTOC=protoc
DOCKER=docker

GIT_COMMIT ?= $(shell git rev-parse --short HEAD || echo unknown)

default: all

all: build test
.PHONY: all

vendor:
	$(GO) mod tidy && $(GO) mod vendor
.PHONY: vendor

build:
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
	$(PROTOC) -I./protos --go_out=./pkg/api/pb --go_opt=paths=source_relative --go-grpc_out=./pkg/api/pb --go-grpc_opt=paths=source_relative protos/api.proto
.PHONY: protos

run:
	$(GO) run cmd/haiku-api/*.go
.PHONY: run

print-%  : ; @echo $* = $($*)

help:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
.PHONY: help

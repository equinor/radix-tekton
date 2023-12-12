ENVIRONMENT ?= dev
VERSION 	?= latest

DNS_ZONE = dev.radix.equinor.com
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
TAG_SAFE_BRANCH_NAME := $(shell echo ${BRANCH} | tr '/' '_')

# If you want to escape branch-environment constraint, pass in OVERRIDE_BRANCH=true

ifeq ($(ENVIRONMENT),prod)
	IS_PROD = yes
else
	IS_DEV = yes
endif

ifdef IS_DEV
	VERSION = dev
endif

ifdef IS_PROD
	DNS_ZONE = radix.equinor.com
endif

CONTAINER_REPO ?= radix$(ENVIRONMENT)
DOCKER_REGISTRY	?= $(CONTAINER_REPO).azurecr.io
HASH := $(shell git rev-parse HEAD)

CLUSTER_NAME = $(shell kubectl config get-contexts | grep '*' | tr -s ' ' | cut -f 3 -d ' ')

TAG := $(TAG_SAFE_BRANCH_NAME)-$(HASH)
BRANCH_TAG := $(TAG_SAFE_BRANCH_NAME)-$(VERSION)

.PHONY: echo
echo:
	@echo "ENVIRONMENT : " $(ENVIRONMENT)
	@echo "DNS_ZONE : " $(DNS_ZONE)
	@echo "CONTAINER_REPO : " $(CONTAINER_REPO)
	@echo "DOCKER_REGISTRY : " $(DOCKER_REGISTRY)
	@echo "BRANCH : " $(BRANCH)
	@echo "CLUSTER_NAME : " $(CLUSTER_NAME)
	@echo "IS_PROD : " $(IS_PROD)
	@echo "IS_DEV : " $(IS_DEV)
	@echo "VERSION : " $(VERSION)
	@echo "TAG : " $(TAG)
	@echo "BRANCH_TAG : " $(BRANCH_TAG)


.PHONY: test
test:
	go test -cover `go list ./... | grep -v 'pkg/client'`

.PHONY: build
build:
	docker build -t $(DOCKER_REGISTRY)/radix-tekton:$(VERSION) -t $(DOCKER_REGISTRY)/radix-tekton:$(BRANCH_TAG) -t $(DOCKER_REGISTRY)/radix-tekton:$(TAG) -f Dockerfile .

.PHONY: deploy
deploy: build
	az acr login --name $(CONTAINER_REPO)
	docker push $(DOCKER_REGISTRY)/radix-tekton:$(BRANCH_TAG)
	docker push $(DOCKER_REGISTRY)/radix-tekton:$(VERSION)
	docker push $(DOCKER_REGISTRY)/radix-tekton:$(TAG)

.PHONY: mocks
mocks:
	mockgen -source ./pkg/models/env/env.go -destination ./pkg/models/env/env_mock.go -package env
	mockgen -source ./pkg/internal/wait/pipelinerun.go -destination ./pkg/internal/wait/pipelinerun_mock.go -package wait

.PHONY: staticcheck
staticcheck:
	staticcheck ./... && go vet ./...

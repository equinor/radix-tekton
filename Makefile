ENVIRONMENT ?= dev
VERSION 	?= latest

DNS_ZONE = dev.radix.equinor.com
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

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

TAG := $(BRANCH)-$(HASH)

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

.PHONY: test
test:	
	go test -cover `go list ./... | grep -v 'pkg/client'`

.PHONY: build
build:
	docker build -t $(DOCKER_REGISTRY)/radix-tekton:$(VERSION) -t $(DOCKER_REGISTRY)/radix-tekton:$(BRANCH)-$(VERSION) -t $(DOCKER_REGISTRY)/radix-tekton:$(TAG) -f Dockerfile .

.PHONY: deploy
deploy:
	az acr login --name $(CONTAINER_REPO)
	make build
	docker push $(DOCKER_REGISTRY)/radix-tekton:$(BRANCH)-$(VERSION)
	docker push $(DOCKER_REGISTRY)/radix-tekton:$(VERSION)
	docker push $(DOCKER_REGISTRY)/radix-tekton:$(TAG)

.PHONY: staticcheck
staticcheck:
	staticcheck ./... && go vet ./...

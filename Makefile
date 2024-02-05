################################################################################
##                             VERSION PARAMS                                 ##
################################################################################

## Docker Build Versions
DOCKER_BUILD_IMAGE = golang:1.20
DOCKER_BASE_IMAGE = alpine:3.19

################################################################################

GO ?= $(shell command -v go 2> /dev/null)
MATTERMOST_ONCALL_HANDLER_IMAGE_REPO ?=stylianosrigas/mattermost-oncall-handler
MATTERMOST_ONCALL_HANDLER_IMAGE ?= stylianosrigas/mattermost-oncall-handler:test
MACHINE = $(shell uname -m)
GOFLAGS ?= $(GOFLAGS:)
BUILD_TIME := $(shell date -u +%Y%m%d.%H%M%S)
BUILD_HASH := $(shell git rev-parse HEAD)

################################################################################
TOOLS_BIN_DIR := $(abspath bin)
GO_INSTALL = ./scripts/go_install.sh
ENSURE_GOLANGCI_LINT = ./scripts/ensure_golangci-lint.sh

OUTDATED_VER := master
OUTDATED_BIN := go-mod-outdated
OUTDATED_GEN := $(TOOLS_BIN_DIR)/$(OUTDATED_BIN)

LOGRUS_URL := github.com/sirupsen/logrus

LOGRUS_VERSION := $(shell find go.mod -type f -exec cat {} + | grep ${LOGRUS_URL} | awk '{print $$NF}')

LOGRUS_PATH := $(GOPATH)/pkg/mod/${LOGRUS_URL}\@${LOGRUS_VERSION}

export GO111MODULE=on

all: check-style dist

## Checks if files are formatted with go fmt.
.PHONY: goformat
goformat:
	@echo Checking if code is formatted
	@for package in $(PACKAGES); do \
		echo "Checking "$$package; \
		files=$$(go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $$package); \
		if [ "$$files" ]; then \
			gofmt_output=$$(gofmt -d -s $$files 2>&1); \
			if [ "$$gofmt_output" ]; then \
				echo "$$gofmt_output"; \
				echo "gofmt failed"; \
				echo "To fix it, run:"; \
				echo "go fmt [FAILED_PACKAGE]"; \
				exit 1; \
			fi; \
		fi; \
	done
	@echo "gofmt success"; \

## Checks if imports are formatted correctly.
.PHONY: goimports
goimports: $(GOIMPORTS)
	@echo Checking if imports are sorted
	@for package in $(PACKAGES); do \
		echo "Checking "$$package; \
		files=$$(go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $$package); \
		if [ "$$files" ]; then \
			goimports_output=$$($(GOIMPORTS) -d $$files 2>&1); \
			if [ "$$goimports_output" ]; then \
				echo "$$goimports_output"; \
				echo "goimports failed"; \
				echo "To fix it, run:"; \
				echo "goimports -w [FAILED_PACKAGE]"; \
				exit 1; \
			fi; \
		fi; \
	done
	@echo "goimports success"; \

## Runs govet and gofmt against all packages.
.PHONY: check-style
check-style: govet lint goformat goimports
	@echo Checking for style guide compliance

## Runs lint against all packages.
.PHONY: lint
lint:
	@echo Running lint
	env GO111MODULE=off $(GO) get -u golang.org/x/lint/golint
	golint -set_exit_status ./...
	@echo lint success

## Runs govet against all packages.
.PHONY: vet
govet:
	@echo Running govet
	$(GO) vet ./...
	@echo Govet success

## Builds and thats all :)
.PHONY: dist
dist:	build

.PHONY: binaries
binaries: ## Build binaries of mattermost-oncall-handler
	@echo Building binaries of mattermost-oncall-handler
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/mattermost-oncall-handler-linux-amd64  ./
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/mattermost-oncall-handler-darwin-amd64  ./
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/mattermost-oncall-handler-linux-arm64  ./
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/mattermost-oncall-handler-darwin-arm64  ./

.PHONY: build
build:
	@echo Building mattermost-oncall-handler
	@if [ "$(ARCH)" = "amd64" ]; then \
		export GOARCH="amd64"; \
	elif [ "$(ARCH)" = "arm64" ]; then \
		export GOARCH="arm64"; \
	elif [ "$(ARCH)" = "arm" ]; then \
		export GOARCH="arm"; \
	else \
		echo "Unknown architecture $(ARCH)"; \
		exit 1; \
	fi; \
	GOOS=linux CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/main  ./

.PHONY: build-image
build-image:  ## Build the docker image for mattermost-oncall-handler
	@echo Building mattermost-oncall-handler Docker Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
	--platform linux/amd64,linux/arm64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile -t $(MATTERMOST_ONCALL_HANDLER_IMAGE) \
	--no-cache \
	--push

.PHONY: build-image-with-tag
build-image-with-tag:  ## Build the docker image for mattermost-oncall-handler
	@echo Building mattermost-oncall-handler Docker Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
	--platform linux/amd64,linux/arm64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile -t $(MATTERMOST_ONCALL_HANDLER_IMAGE) -t $(MATTERMOST_ONCALL_HANDLER_IMAGE_REPO):${TAG} \
	--push

.PHONY: push-image-pr
push-image-pr:
	@echo Push Image PR
	bash ./scripts/push-image-pr.sh

.PHONY: push-image
push-image:
	@echo Push Image
	bash ./scripts/push-image.sh

.PHONY: install
install: build
	go install ./...

.PHONY: release
release:
	@echo Cut a release
	bash ./scripts/release.sh

.PHONY: deps
deps:
	sudo apt update && sudo apt install hub git
	go get k8s.io/release/cmd/release-notes

.PHONY: check-modules
check-modules: $(OUTDATED_GEN) ## Check outdated modules
	@echo Checking outdated modules
	$(GO) list -mod=mod -u -m -json all | $(OUTDATED_GEN) -update -direct


$(OUTDATED_GEN): ## Build go-mod-outdated.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/psampaz/go-mod-outdated $(OUTDATED_BIN) $(OUTDATED_VER)

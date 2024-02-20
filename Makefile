export DOCKER_BUILDKIT=1
LDFLAG_LOCATION := github.com/metal-toolbox/bomservice/internal/version
GIT_COMMIT  := $(shell git rev-parse --short HEAD)
GIT_BRANCH  := $(shell git symbolic-ref -q --short HEAD)
GIT_SUMMARY := $(shell git describe --tags --dirty --always)
VERSION     := $(shell git describe --tags 2> /dev/null)
BUILD_DATE  := $(shell date +%s)
GIT_COMMIT_FULL  := $(shell git rev-parse HEAD)
GO_VERSION := $(shell expr `go version |cut -d ' ' -f3 |cut -d. -f2` \>= 16)
DOCKER_IMAGE  := "ghcr.io/metal-toolbox/bomservice"
REPO := "https://github.com/metal-toolbox/bomservice.git"

.DEFAULT_GOAL := help

## lint
lint:
	golangci-lint run

## Go test
test: lint
	CGO_ENABLED=0 go test -timeout 1m -v -covermode=atomic ./...


## generate mock store - invoke when changes are made to the store interface
gen-store-mock:
	go install github.com/golang/mock/mockgen@v1.6.0
	mockgen -package=mock -source=internal/store/interface.go > internal/store/mock/mock.go

## Build linux bin
build:
ifeq ($(GO_VERSION), 0)
	$(error build requies go version 1.20.x+ or higher)
endif
	GOOS=linux GOARCH=amd64 go build -o bomservice \
	   -ldflags \
		"-X $(LDFLAG_LOCATION).GitCommit=$(GIT_COMMIT) \
         -X $(LDFLAG_LOCATION).GitBranch=$(GIT_BRANCH) \
         -X $(LDFLAG_LOCATION).GitSummary=$(GIT_SUMMARY) \
         -X $(LDFLAG_LOCATION).AppVersion=$(VERSION) \
         -X $(LDFLAG_LOCATION).BuildDate=$(BUILD_DATE)"


## build docker image and tag as ghcr.io/metal-toolbox/bomservice:latest
build-image: build
	@echo ">>>> NOTE: You may want to execute 'make build-image-nocache' depending on the Docker stages changed"
	docker build --rm=true -f Dockerfile -t ${DOCKER_IMAGE}:latest  . \
							 --label org.label-schema.schema-version=1.0 \
							 --label org.label-schema.vcs-ref=$(GIT_COMMIT_FULL) \
							 --label org.label-schema.vcs-url=$(REPO)

## build and push devel docker image to KIND image repo
push-image-devel: build-image
	docker tag ${DOCKER_IMAGE}:latest localhost:5001/bomservice:latest
	docker push localhost:5001/bomservice:latest
	kind load docker-image localhost:5001/bomservice:latest

## push docker image
push-image:
	docker push ${DOCKER_IMAGE}:latest


# https://gist.github.com/prwhite/8168133
# COLORS
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)


TARGET_MAX_CHAR_NUM=20
## Show help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  ${YELLOW}%-$(TARGET_MAX_CHAR_NUM)s${RESET} ${GREEN}%s${RESET}\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

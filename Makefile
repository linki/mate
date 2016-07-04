.PHONY: clean clean check docker scm-source

BINARY        ?= mate
VERSION       ?= $(shell git describe --tags --always --dirty)
IMAGE         ?= pierone.stups.zalan.do/teapot/$(BINARY)
TAG           ?= $(VERSION)
GITHEAD       = $(shell git rev-parse --short HEAD)
GITURL        = $(shell git config --get remote.origin.url)
GITSTATUS     = $(shell git status --porcelain || echo "no changes")
BUILD_FLAGS   ?= -v
LDFLAGS       ?= -X main.version=$(VERSION)

default: build.local

clean:
	rm -rf build

check:
	golint ./... | egrep -v '^vendor/'
	go vet -v ./... 2>&1 | egrep -v '^(vendor/|exit status 1)'

prepare:
	mkdir -p build/linux
	mkdir -p build/osx

build.local: prepare $(wildcard *.go) $(wildcard */*.go)
	go build -o build/"$(BINARY)" "$(BUILD_FLAGS)" -ldflags "$(LDFLAGS)" .

build.linux: prepare $(wildcard *.go) $(wildcard */*.go)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build "$(BUILD_FLAGS)" -o build/linux/"$(BINARY)" -ldflags "$(LDFLAGS)" .

build.osx: prepare $(wildcard *.go) $(wildcard */*.go)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build "$(BUILD_FLAGS)" -o build/osx/"$(BINARY)" -ldflags "$(LDFLAGS)" .

build.docker: scm-source build.linux
	docker build -t "$(IMAGE):$(TAG)" .

build.push: build.docker
	docker push "$(IMAGE):$(TAG)"

scm-source:
	@echo "{\"url\": \"$(GITURL)\", \"revision\": \"$(GITHEAD)\", \"status\": \"$(GITSTATUS)\"}" > scm-source.json

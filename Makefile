BINARY        ?= mate
VERSION       ?= $(shell git describe --tags --dirty)
IMAGE         ?= pierone.stups.zalan.do/teapot/mate
TAG           ?= $(VERSION)
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

build.local: prepare
	go build -o build/"$(BINARY)" "$(BUILD_FLAGS)" -ldflags "$(LDFLAGS)" .

build.linux: prepare
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build "$(BUILD_FLAGS)" -o build/linux/"$(BINARY)" -ldflags "$(LDFLAGS)" .

build.osx: prepare
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build "$(BUILD_FLAGS)" -o build/osx/"$(BINARY)" -ldflags "$(LDFLAGS)" .

build.docker:
	docker build -t "$(IMAGE):$(TAG)" .

build.push: build.docker
	docker push "$(IMAGE):$(TAG)"

.PHONY: clean check build.local build.linux build.osx build.docker build.push

BINARY        		 ?= mate
VERSION       		 ?= $(shell git describe --tags --always --dirty)
IMAGE         		 ?= registry-write.opensource.zalan.do/teapot/$(BINARY)
TAG           		 ?= $(VERSION)
GITHEAD       		 = $(shell git rev-parse --short HEAD)
GITURL        		 = $(shell git config --get remote.origin.url)
GITSTATUS     		 = $(shell git status --porcelain || echo "no changes")
SOURCES       		 = $(shell find . -name '*.go')
DOCKERFILE    		 ?= Dockerfile
DOCKERFILE_ZALANDO ?= Dockerfile.zalando
GOPKGS        		 = $(shell go list ./... | grep -v /vendor/)
BUILD_FLAGS   		 ?= -v
LDFLAGS       		 ?= -X main.version=$(VERSION) -w -s

default: build.local

clean:
	rm -rf build

test:
	go test -v $(GOPKGS)

test-with-aws:
	AWS_PROVIDER_TEST=true \
	AWS_HOSTED_ZONE=mate.teapot.zalan.do \
	go test -v $(GOPKGS)

fmt:
	go fmt $(GOPKGS)

check:
	golint $(GOPKGS)
	go vet -v $(GOPKGS)

build.local: build/$(BINARY)
build.linux: build/linux/$(BINARY)
build.osx: build/osx/$(BINARY)

build/$(BINARY): $(SOURCES)
	go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

build/linux/$(BINARY): $(SOURCES)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/linux/$(BINARY) -ldflags "$(LDFLAGS)" .

build/osx/$(BINARY): $(SOURCES)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/osx/$(BINARY) -ldflags "$(LDFLAGS)" .

$(DOCKERFILE_ZALANDO).upstream: $(DOCKERFILE_ZALANDO)
	sed "s@UPSTREAM@$(shell $(shell head -1 $(DOCKERFILE_ZALANDO) | sed -E 's@FROM (.*)/(.*)/(.*):.*@pierone latest \2 \3 --url \1@'))@" $(DOCKERFILE_ZALANDO) > $(DOCKERFILE_ZALANDO).upstream

build.docker.zalando: $(DOCKERFILE_ZALANDO).upstream scm-source.json build.linux #zalando compliant image
	docker build --rm -t "$(IMAGE):$(TAG)" -f $(DOCKERFILE_ZALANDO).upstream .
	
build.push: build.docker
	docker push "$(IMAGE):$(TAG)"

build.docker: build.linux #using hub alpine
	docker build --rm -t "$(IMAGE):$(TAG)" -f $(DOCKERFILE) .

scm-source.json: .git
	@echo '{"url": "$(GITURL)", "revision": "$(GITHEAD)", "author": "$(USER)", "status": "$(GITSTATUS)"}' > scm-source.json

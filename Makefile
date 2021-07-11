.PHONY: all test build-binary image clean

GO ?= go
GO_CMD := CGO_ENABLED=0 $(GO)
GIT_VERSION := $(shell git describe --tags --dirty --always)
VERSION := $(GIT_VERSION:v%=%)
GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_BRANCH := $(shell git branch --show-current)
DOCKER_TAG != if [ "$(GIT_BRANCH)" = "master" ]; then \
		echo "latest"; \
	else \
		echo "$(VERSION)"; \
	fi

all: test build-binary

test:
	$(GO_CMD) test -cover ./...

build-binary:
	$(GO_CMD) build -tags netgo -ldflags "-w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT)" -o steam-exporter .

image:
	docker build -t "ghcr.io/xperimental/steam-exporter:$(DOCKER_TAG)" .

all-images:
	docker buildx build -t "ghcr.io/xperimental/steam-exporter:$(DOCKER_TAG)" --platform linux/amd64,linux/arm64 --push .

clean:
	rm -f steam-exporter

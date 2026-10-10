APP_NAME := opencode2api
IMAGE ?= ghcr.io/moewsama/opencode2api:latest
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test vet docker-build clean

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/opencode2api

test:
	go test ./...

vet:
	go vet ./...

# The Dockerfile compiles the binary itself (multi-stage build), so no local
# artifact is required first. Single-arch on purpose: the published image is
# linux/amd64 only.
docker-build:
	docker build --platform linux/amd64 --build-arg VERSION=$(VERSION) -t $(IMAGE) .

clean:
	rm -rf bin

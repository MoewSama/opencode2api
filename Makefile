APP_NAME := opencode2api
IMAGE ?= ghcr.io/moewsama/opencode2api:latest
VERSION ?= dev

.PHONY: build docker-build

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/$(APP_NAME) ./

docker-build: build
	test -f bin/$(APP_NAME)
	docker build --platform linux/amd64 --build-arg VERSION=$(VERSION) -t $(IMAGE) .

#!/usr/bin/make -f

export CGO_ENABLED=0

IMAGE=skpr/fluentbit-cloudwatchlogs
VERSION=$(shell git describe --tags --always)

# Builds the project.
build:
	docker build  -t ${IMAGE}:${VERSION} -t ${IMAGE}:latest .

# Run all lint checking with exit codes for CI.
lint:
	golint -set_exit_status `go list ./... | grep -v /vendor/`

# Run tests with coverage reporting.
test:
	go test -cover ./...

release: build push

push:
	docker push ${IMAGE}:${VERSION}
	docker push ${IMAGE}:latest

.PHONY: *

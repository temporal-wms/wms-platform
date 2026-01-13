GOCACHE ?= $(CURDIR)/.gocache

.PHONY: test build coverage coverage-html

test:
	GOCACHE=$(GOCACHE) go test ./...

build:
	GOCACHE=$(GOCACHE) go build ./...

coverage:
	GOCACHE=$(GOCACHE) go test ./... -coverprofile=cover.out
	GOCACHE=$(GOCACHE) go tool cover -func=cover.out

coverage-html:
	GOCACHE=$(GOCACHE) go test ./... -coverprofile=cover.out
	GOCACHE=$(GOCACHE) go tool cover -html=cover.out -o coverage.html

HELM_HOME ?= $(shell helm home)
HELM_PLUGIN_DIR ?= $(HELM_HOME)/plugins/helm-whatup
HAS_GLIDE := $(shell command -v glide;)
VERSION := $(shell sed -n -e 's/version:[ "]*\([^"]*\).*/\1/p' plugin.yaml)
DIST := $(CURDIR)/_dist
LDFLAGS := "-X main.version=${VERSION}"

.PHONY: build
build:
	go build -o bin/helm-whatup -ldflags $(LDFLAGS) ./main.go

.PHONY: dist
dist:
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 go build -o bin/helm-whatup ./main.go
	tar -zcvf $(DIST)/helm-whatup-$(VERSION)-linux-amd64.tar.gz bin/helm-whatup README.md LICENSE.md plugin.yaml
	GOOS=darwin GOARCH=amd64 go build -o bin/helm-whatup ./main.go
	tar -zcvf $(DIST)/helm-whatup-$(VERSION)-darwin-amd64.tar.gz bin/helm-whatup README.md LICENSE.md plugin.yaml


.PHONY: bootstrap
bootstrap:
	glide install
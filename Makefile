HELM_HOME ?= $(shell helm home)
HELM_PLUGIN_DIR ?= $(HELM_HOME)/plugins/helm-whatup
HAS_GLIDE := $(shell command -v glide;)
VERSION := $(shell sed -n -e 's/version:[ "]*\([^"]*\).*/\1/p' plugin.yaml)
DIST := $(CURDIR)/_dist
LDFLAGS := "-X main.version=${VERSION}"
HELM_FLAG := --host 127.0.0.1:44134

.PHONY: helmrel
helmrel:
	helm repo update
	helm $(HELM_FLAG) install -n coredns --version 1.5.0 stable/coredns
	helm $(HELM_FLAG) install -n jenkins --version 0.32.1 stable/jenkins
	helm $(HELM_FLAG) install -n kafka-manager --version 1.1.1 stable/kafka-manager
	helm $(HELM_FLAG) install -n kapacitor --version 0.3.0 stable/kapacitor
	helm $(HELM_FLAG) install -n hunter --version 1.1.5 stable/karma
	helm $(HELM_FLAG) install -n kube-hunter --version 1.0.0 stable/kube-hunter
	helm $(HELM_FLAG) install -n kube-slack --version 0.1.0 stable/kube-slack
	helm $(HELM_FLAG) install -n kuberhealthy --version 1.1.1 stable/kuberhealthy
	helm $(HELM_FLAG) install -n lamp --version 0.1.2 stable/lamp
	helm $(HELM_FLAG) install -n luigi --version 2.7.2 stable/luigi
	helm $(HELM_FLAG) install -n magento --version 0.4.10 stable/magento

.PHONY: test
test: build
	go test ./...

.PHONY: cov
cov: build
	go test -coverprofile c.out ./...

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
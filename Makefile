#
# Makefile for Priam
#
# Copyright (C) 2016 VMware, Inc.  All rights reserved.
# -- VMware Confidential
#

.PHONY: all check govet update-build-dependencies build-testaid generate-mocks build test coverage cover install help 

# GO program
GO=go

# Default target is all
_default: all

all: check build test

check: govet

govet:
	@echo checking go vet...
	$(GO) tool vet -structtags=false -methods=false .

update-build-dependencies:
	$(GO) get

build: update-build-dependencies
	@echo building...
	$(GO) build

build-testaid:
	$(GO) get ./testaid
	$(GO) install ./testaid

generate-mocks: build-testaid
	$(GO) get github.com/vektra/mockery/.../
	$(GOPATH)/bin/mockery -dir=core -name=DirectoryService
	$(GOPATH)/bin/mockery -dir=core -name=ApplicationService
	$(GOPATH)/bin/mockery -dir=core -name=AppTemplateService

test: update-build-dependencies generate-mocks
	@echo testing...
	$(GO) test -cover ./util ./core ./cli

coverage: update-build-dependencies generate-mocks
	@echo generating test coverage report...
	$(GO) test -coverprofile=util.cover.out ./util
	$(GO) test -coverprofile=core.cover.out -coverpkg=./util,./core ./core
	$(GO) test -coverprofile=cli.cover.out -coverpkg=./util,./core,./cli ./cli
	echo "mode: set" > coverage.out
	cat *.cover.out | grep -v mode: | sort -r | awk '{if($$1 != last) {print $$0;last=$$1}}' >> coverage.out
	rm util.cover.out core.cover.out cli.cover.out
	$(GO) tool cover -html=coverage.out

cover: coverage

# We will probaby have to run "go install github.com/vmware/priam" when ready
install:
	$(GO) install

help:
	@echo 'Priam Makefile help'
	@echo
	@echo 'Targets:'
	@echo '   help          - print this help'
	@echo '   all           - build and test priam'
	@echo '   test          - run priam tests'
	@echo '   install       - install the priam binary'
	@echo '   coverage      - test and open coverage result in the browser'


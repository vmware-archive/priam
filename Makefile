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
	$(GO) vet .

build:
	@echo building...
	$(GO) build

build-testaid:
	$(GO) get ./testaid
	$(GO) install ./testaid

generate-mocks: build-testaid
	$(GO) get github.com/vektra/mockery/cmd/mockery
	rm -f mocks/*.go
	$(GOPATH)/bin/mockery -dir=core -name=DirectoryService
	$(GOPATH)/bin/mockery -dir=core -name=ApplicationService
	$(GOPATH)/bin/mockery -dir=core -name=OauthResource
	$(GOPATH)/bin/mockery -dir=core -name=TokenGrants
	$(GOPATH)/bin/mockery -dir=core -name=TokenServiceFactory

test: generate-mocks
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

install:
	$(GO) install

# since priam packages are installed as libraries in the golang pkg directory we must 
# specifially clean each package with the -i (installed) option. 
clean:
	$(GO) clean -i . ./cli ./core ./mocks ./testaid ./util
	rm -rf mocks/*

help:
	@echo 'Priam Makefile help'
	@echo
	@echo 'Targets:'
	@echo '   help          - print this help'
	@echo '   all           - build and test priam'
	@echo '   clean         - clean go objects'
	@echo '   test          - run priam tests'
	@echo '   install       - install the priam binary'
	@echo '   coverage      - test and open coverage result in the browser'


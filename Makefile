#
# Makefile for Priam
#
# Copyright (C) 2016 VMware, Inc.  All rights reserved.
# -- VMware Confidential
#

.PHONY: test build generate-mocks help

# GO program
GO=go

# Default target is all
_default: all

all: check test

check: govet

govet:
	@echo checking go vet...
	$(GO) tool vet -structtags=false -methods=false .

build-testaid:
	$(GO) install ./testaid

generate-mocks: build-testaid
	$(GO) get github.com/vektra/mockery/.../
	$(GOPATH)/bin/mockery -dir=core -name=DirectoryService
	$(GOPATH)/bin/mockery -dir=core -name=ApplicationService

build:
	@echo building...
	$(GO) get
	$(GO) build

test: build generate-mocks
	@echo testing...
	$(GO) get github.com/wadey/gocovmerge
	$(GOPATH)/bin/gotestcover -coverprofile=coverage.out ./util ./core ./cli

coverage: test
	$(GO) tool cover -html=coverage.out

# We will probaby have to run "go install github.com/vmware/priam"
# when ready
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


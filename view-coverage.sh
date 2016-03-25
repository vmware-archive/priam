#! /bin/bash

go get github.com/vektra/mockery/.../
go get
$GOPATH/bin/mockery -inpkg -all

go test ./core -coverprofile=coverage.out
go tool cover -html=coverage.out

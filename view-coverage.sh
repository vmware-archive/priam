#! /bin/bash
set -e
cd "$( dirname "${BASH_SOURCE[0]}" )"

go get github.com/vektra/mockery/.../
go get
$GOPATH/bin/mockery -inpkg -all

go test ./core -coverprofile=coverage.out
go tool cover -html=coverage.out

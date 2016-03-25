#! /bin/bash


go get
$GOPATH/bin/mockery -inpkg -all

go test ./core -coverprofile=coverage.out
go tool cover -html=coverage.out

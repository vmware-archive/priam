#! /bin/bash

set -ev
go build
cp wks cf-wks
cf uninstall-plugin cf-wks
cf install-plugin cf-wks
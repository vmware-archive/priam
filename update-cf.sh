#! /bin/bash

set -ev
go build
cp priam cf-priam
cf uninstall-plugin cf-priam || echo "Ignoring..."
cf install-plugin cf-priam

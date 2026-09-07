#!/usr/bin/env bash
# mise description="Tidy go modules"
set -e

go mod tidy
# go mod vendor

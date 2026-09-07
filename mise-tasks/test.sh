#!/usr/bin/env bash
# mise description="Run tests"
set -e

go test -v ./... -cover -race

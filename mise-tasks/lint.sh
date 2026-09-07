#!/usr/bin/env bash
# mise description="Run linters"
set -e

golangci-lint run --verbose --color always .

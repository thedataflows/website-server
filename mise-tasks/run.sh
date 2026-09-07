#!/usr/bin/env bash
# mise description="Run the app in development mode"
set -e

mise run mailpit-start
go run . "$@"

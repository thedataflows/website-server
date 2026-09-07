#!/usr/bin/env bash
# mise description="Build the app using goreleaser"
set -e

EXE=""
case "$(uname -s)" in
  MINGW*|MSYS*|CYGWIN*|Windows_NT) EXE=".exe" ;;
esac

goreleaser build --clean --snapshot --single-target --id "." --output "./bin/ws${EXE}"

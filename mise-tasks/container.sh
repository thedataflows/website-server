#!/usr/bin/env bash
# mise description="Build container image"
set -e

APP=$(grep module go.mod | cut -d/ -f2)
podman build -t "${APP}:latest" .

#!/usr/bin/env bash
# mise description="Stop mailpit"
set -e

PID_FILE="bin/mailpit.pid"
PID=$(cat "$PID_FILE" 2>/dev/null || true)
if [[ -z "$PID" ]] || ! ps -p "$PID" >/dev/null 2>&1; then
  echo "mailpit is not running"
  rm -f "$PID_FILE"
  exit 0
fi

echo "Stopping mailpit (PID $PID)"
kill "$PID" && rm -f "$PID_FILE"

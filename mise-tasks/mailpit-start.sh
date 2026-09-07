#!/usr/bin/env bash
# mise description="Start mailpit in the background"
set -e

mkdir -p bin
PID_FILE="bin/mailpit.pid"
PID=$(cat "$PID_FILE" 2>/dev/null || true)
if [[ -n "$PID" ]] && ps -p "$PID" >/dev/null 2>&1; then
  echo "mailpit already running (PID $PID)"
  exit 0
fi

echo "Starting mailpit in the background."
nohup mailpit --verbose --db-file bin/mailpit.db >bin/mailpit.log 2>&1 &
echo "$!" > "$PID_FILE"
sleep 1
echo "PID $(cat "$PID_FILE")"

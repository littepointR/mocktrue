#!/bin/bash
# vserial_loopback.sh - Test serial loopback using socat virtual pair.
# This script verifies the basic send/receive functionality.
# Usage: ./vserial_loopback.sh

set -e

PORT1="/tmp/ttyV0"
PORT2="/tmp/ttyV1"

# Clean up old symlinks
rm -f "$PORT1" "$PORT2"

echo "=== Serial Loopback Test ==="
echo "Starting socat virtual pair..."

# Start socat in background
socat -d -d \
  pty,raw,echo=0,link="$PORT1" \
  pty,raw,echo=0,link="$PORT2" &
SOCAT_PID=$!

# Wait for symlinks to appear
for i in {1..20}; do
  if [ -e "$PORT1" ] && [ -e "$PORT2" ]; then
    break
  fi
  sleep 0.1
done

if [ ! -e "$PORT1" ] || [ ! -e "$PORT2" ]; then
  echo "FAIL: socat symlinks not created"
  kill $SOCAT_PID 2>/dev/null || true
  exit 1
fi

echo "Virtual pair created: $PORT1 <-> $PORT2"

# Test 1: Write to PORT1, read from PORT2
echo "Test 1: Write 'hello' to $PORT1, read from $PORT2"
echo -n "hello" > "$PORT1" &
sleep 0.5
RECEIVED=$(cat "$PORT2" &
CAT_PID=$!
sleep 1
kill $CAT_PID 2>/dev/null || true)
wait $CAT_PID 2>/dev/null || true

if [ "$RECEIVED" = "hello" ]; then
  echo "PASS: Received 'hello'"
else
  echo "FAIL: Expected 'hello', got '$RECEIVED'"
  kill $SOCAT_PID 2>/dev/null || true
  exit 1
fi

# Test 2: Write to PORT2, read from PORT1
echo "Test 2: Write 'world' to $PORT2, read from $PORT1"
echo -n "world" > "$PORT2" &
sleep 0.5
RECEIVED=$(cat "$PORT1" &
CAT_PID=$!
sleep 1
kill $CAT_PID 2>/dev/null || true)
wait $CAT_PID 2>/dev/null || true

if [ "$RECEIVED" = "world" ]; then
  echo "PASS: Received 'world'"
else
  echo "FAIL: Expected 'world', got '$RECEIVED'"
  kill $SOCAT_PID 2>/dev/null || true
  exit 1
fi

echo "=== All loopback tests passed ==="

# Cleanup
kill $SOCAT_PID 2>/dev/null || true
rm -f "$PORT1" "$PORT2"

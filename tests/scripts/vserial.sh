#!/bin/bash
# vserial.sh - Start a socat virtual serial pair for testing.
# Usage: ./vserial.sh
# Creates /tmp/ttyV0 and /tmp/ttyV1 as a virtual null-modem pair.
# Stop with Ctrl+C or kill the socat process.

set -e

# Remove old symlinks if they exist
rm -f /tmp/ttyV0 /tmp/ttyV1

echo "Starting socat virtual serial pair..."
echo "  /tmp/ttyV0 <-> /tmp/ttyV1"
echo "Press Ctrl+C to stop."

socat -d -d \
  pty,raw,echo=0,link=/tmp/ttyV0 \
  pty,raw,echo=0,link=/tmp/ttyV1

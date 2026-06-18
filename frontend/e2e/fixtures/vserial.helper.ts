import { execSync, spawn, ChildProcess } from 'child_process';
import { existsSync } from 'fs';

export interface VirtualPair {
  port1: string;
  port2: string;
  process: ChildProcess | null;
}

/**
 * Start a socat virtual serial pair for testing.
 * Returns the two port paths and the socat process.
 */
export function startVirtualPair(): VirtualPair {
  const port1 = '/tmp/ttyV0';
  const port2 = '/tmp/ttyV1';

  // Remove old symlinks
  try {
    execSync(`rm -f ${port1} ${port2}`);
  } catch (e) {
    // Ignore errors
  }

  // Start socat in background
  const socat = spawn('socat', [
    '-d', '-d',
    'pty,raw,echo=0,link=/tmp/ttyV0',
    'pty,raw,echo=0,link=/tmp/ttyV1',
  ], {
    stdio: 'pipe',
    detached: true,
  });

  // Wait for symlinks to appear
  const deadline = Date.now() + 2000;
  while (Date.now() < deadline) {
    if (existsSync(port1) && existsSync(port2)) {
      return { port1, port2, process: socat };
    }
    execSync('sleep 0.05');
  }

  socat.kill();
  throw new Error('Timeout waiting for socat symlinks');
}

/**
 * Stop a virtual pair and clean up.
 */
export function stopVirtualPair(pair: VirtualPair) {
  if (pair.process) {
    pair.process.kill();
    try {
      execSync(`rm -f ${pair.port1} ${pair.port2}`);
    } catch (e) {
      // Ignore cleanup errors
    }
  }
}

/**
 * Write data to a serial port (used for testing).
 */
export function writeToPort(portPath: string, data: string) {
  execSync(`echo -n "${data}" > ${portPath}`);
}

/**
 * Read data from a serial port (non-blocking, returns what's available).
 */
export function readFromPort(portPath: string, timeoutMs: number = 1000): string {
  try {
    return execSync(`timeout ${timeoutMs / 1000} cat ${portPath} 2>/dev/null || true`, {
      encoding: 'utf-8',
      timeout: timeoutMs + 500,
    }).trim();
  } catch (e) {
    return '';
  }
}

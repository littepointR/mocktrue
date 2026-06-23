import { execFileSync, spawn, ChildProcess } from 'child_process';
import { existsSync } from 'fs';
import { writeFileSync } from 'fs';
import { tmpdir } from 'os';
import path from 'path';

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
  const suffix = `${process.pid}-${Date.now()}-${Math.floor(Math.random() * 10000)}`;
  const port1 = `/tmp/portweave-${suffix}-a`;
  const port2 = `/tmp/portweave-${suffix}-b`;

  // Remove old symlinks
  try {
    execFileSync('rm', ['-f', port1, port2]);
  } catch (e) {
    // Ignore errors
  }

  // Start socat in background
  const socat = spawn('socat', [
    '-d', '-d',
    `pty,raw,echo=0,link=${port1}`,
    `pty,raw,echo=0,link=${port2}`,
  ], {
    stdio: 'pipe',
    detached: true,
  });
  let stderr = '';
  socat.stderr?.on('data', chunk => {
    stderr += String(chunk);
  });

  // Wait for symlinks to appear
  const deadline = Date.now() + 2000;
  while (Date.now() < deadline) {
    if (existsSync(port1) && existsSync(port2)) {
      return { port1, port2, process: socat };
    }
    execFileSync('sleep', ['0.05']);
  }

  socat.kill();
  throw new Error(`Timeout waiting for socat symlinks: ${stderr.trim()}`);
}

/**
 * Stop a virtual pair and clean up.
 */
export function stopVirtualPair(pair: VirtualPair) {
  if (pair.process) {
    pair.process.kill();
    try {
      execFileSync('rm', ['-f', pair.port1, pair.port2]);
    } catch (e) {
      // Ignore cleanup errors
    }
  }
}

/**
 * Write data to a serial port (used for testing).
 */
export function writeToPort(portPath: string, data: string) {
  execFileSync('sh', ['-c', 'cat "$1" > "$2"', 'sh', writeTempData(data), portPath]);
}

export function writeBytesToPort(portPath: string, data: Buffer) {
  execFileSync('sh', ['-c', 'cat "$1" > "$2"', 'sh', writeTempData(data), portPath]);
}

/**
 * Read data from a serial port (non-blocking, returns what's available).
 */
export function readFromPort(portPath: string, timeoutMs: number = 1000): string {
  try {
    return execFileSync('sh', ['-c', 'timeout "$1" cat "$2" 2>/dev/null || true', 'sh', String(timeoutMs / 1000), portPath], {
      encoding: 'utf-8',
      timeout: timeoutMs + 500,
    }).trim();
  } catch (e) {
    return '';
  }
}

function writeTempData(data: string | Buffer): string {
  const file = path.join(tmpdir(), `portweave-e2e-${process.pid}-${Date.now()}-${Math.random()}`);
  writeFileSync(file, data);
  return file;
}

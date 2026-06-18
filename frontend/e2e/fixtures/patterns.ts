/**
 * Test patterns and data generators for E2E tests.
 */

/** Generate a known byte pattern for testing. */
export function generatePattern(size: number): Buffer {
  const buf = Buffer.alloc(size);
  for (let i = 0; i < size; i++) {
    buf[i] = i % 256;
  }
  return buf;
}

/** Generate a sequential byte pattern with incrementing values. */
export function generateSequential(size: number, start: number = 0): Buffer {
  const buf = Buffer.alloc(size);
  for (let i = 0; i < size; i++) {
    buf[i] = (start + i) % 256;
  }
  return buf;
}

/** Convert hex string to buffer. */
export function hexToBuffer(hex: string): Buffer {
  return Buffer.from(hex.replace(/\s+/g, ''), 'hex');
}

/** Convert buffer to hex string. */
export function bufferToHex(buf: Buffer): string {
  return Array.from(buf)
    .map(b => b.toString(16).padStart(2, '0'))
    .join(' ');
}

/** Generate a Modbus RTU frame. */
export function generateModbusFrame(slaveAddr: number, funcCode: number, data: Buffer): Buffer {
  const frame = Buffer.alloc(2 + data.length + 2); // addr + func + data + crc16
  frame[0] = slaveAddr;
  frame[1] = funcCode;
  data.copy(frame, 2);

  // Calculate CRC16
  const crc = crc16(frame.slice(0, 2 + data.length));
  frame[2 + data.length] = crc & 0xFF;
  frame[3 + data.length] = (crc >> 8) & 0xFF;

  return frame;
}

/** CRC16 for Modbus RTU. */
function crc16(data: Buffer): number {
  let crc = 0xFFFF;
  for (const byte of data) {
    crc ^= byte;
    for (let i = 0; i < 8; i++) {
      if (crc & 0x0001) {
        crc = (crc >> 1) ^ 0xA001;
      } else {
        crc >>= 1;
      }
    }
  }
  return crc;
}

/** Generate an AA55 custom frame. */
export function generateAA55Frame(cmd: number, data: Buffer): Buffer {
  const payloadLen = data.length + 1; // cmd + data
  const frame = Buffer.alloc(2 + 2 + 1 + data.length); // header + length + checksum + cmd + data
  frame[0] = 0xAA;
  frame[1] = 0x55;
  frame[2] = payloadLen & 0xFF;
  frame[3] = (payloadLen >> 8) & 0xFF;
  frame[4] = cmd;
  data.copy(frame, 5);

  // Calculate sum8 checksum (header + length + cmd + data)
  let sum = 0;
  for (let i = 0; i < 4 + payloadLen; i++) {
    sum = (sum + frame[i]) & 0xFF;
  }
  frame[4 + payloadLen] = sum;

  return frame;
}

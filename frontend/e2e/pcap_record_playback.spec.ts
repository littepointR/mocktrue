import { test } from '@playwright/test';

test.describe.skip('PCAP Record and Playback E2E', () => {
  test('should record data to pcap file', async () => {
    test.skip(true, 'PCAP recording UI is not implemented in the current serial panel');
  });

  test('should playback recorded data', async () => {
    test.skip(true, 'PCAP playback UI is not implemented in the current serial panel');
  });
});

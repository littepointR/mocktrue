import { test } from '@playwright/test';

test.describe.skip('Protocol Parse E2E', () => {
  test('should parse Modbus RTU frame', async () => {
    test.skip(true, 'protocol parser UI is not implemented in the current serial panel');
  });

  test('should parse AA55 custom frame', async () => {
    test.skip(true, 'protocol parser UI is not implemented in the current serial panel');
  });
});

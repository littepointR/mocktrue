import type { Page } from '@playwright/test';

export async function injectWailsMock(page: Page) {
  await page.addInitScript(() => {
    const mockState = {
      ports: [
        { Name: '/tmp/ttyV0', IsUSB: false, VID: '', PID: '', SerialNumber: '', FriendlyName: '/tmp/ttyV0' },
        { Name: '/tmp/ttyV1', IsUSB: false, VID: '', PID: '', SerialNumber: '', FriendlyName: '/tmp/ttyV1' },
        { Name: '/tmp/ttyV2', IsUSB: false, VID: '', PID: '', SerialNumber: '', FriendlyName: '/tmp/ttyV2' },
      ],
      handles: [] as any[],
      virtualPairs: [] as any[],
      virtualPorts: [] as any[],
      bridges: [] as any[],
      graphBufferText: 'hello',
      nextHandleId: 1,
    };

    function utf8Length(text: string): number {
      return new TextEncoder().encode(text).length;
    }

    function hexLength(text: string): number {
      const compact = text.replace(/\s+/g, '');
      if (compact.length % 2 !== 0 || !/^[\da-fA-F]*$/.test(compact)) {
        throw new Error('decode hex content');
      }
      return compact.length / 2;
    }

    function openHandle(req: any) {
      const existing = mockState.handles.find(h =>
        h.IsOpen && h.Config.PortName === req.Config.PortName
      );
      if (existing) {
        return existing;
      }

      const id = `port-${mockState.nextHandleId++}`;
      const handle = {
        ID: id,
        Config: req.Config,
        IsOpen: true,
        RxBytes: 0,
        TxBytes: 0,
      };
      mockState.handles.push(handle);
      return handle;
    }

    const handlers: Record<number, (...args: any[]) => any> = {
      3334354438: (msg: string) => `pong:${msg}`,
      2380440348: () => mockState.ports,
      1403721065: openHandle,
      358686901: (id: string) => {
        const idx = mockState.handles.findIndex(h => h.ID === id);
        if (idx >= 0) mockState.handles.splice(idx, 1);
        return null;
      },
      2409741650: () => mockState.handles,
      257355161: () => ({ Data: [], Total: 0, Offset: 0, Length: 0 }),
      304932660: (req: any) => {
        const handle = mockState.handles.find(h => h.ID === req.PortID);
        const byteLength = req.Mode === 'hex' ? hexLength(req.Content) : utf8Length(req.Content);
        if (handle) handle.TxBytes += byteLength;
        return byteLength;
      },
      4017649972: (id: string) => {
        const handle = mockState.handles.find(h => h.ID === id);
        if (!handle) {
          throw new Error(`handle not found: ${id}`);
        }
        handle.RxBytes = 0;
        handle.TxBytes = 0;
        return null;
      },
      3426122829: (id: string, p1: string, p2: string) => {
        if (mockState.virtualPairs.some(pair => pair.ID === id)) {
          throw new Error('pair ID already exists');
        }
        const pair = { ID: id, Port1: `/tmp/${p1}`, Port2: `/tmp/${p2}` };
        mockState.virtualPairs.push(pair);
        mockState.ports.push(
          { Name: pair.Port1, IsUSB: false, VID: '', PID: '', SerialNumber: '', FriendlyName: pair.Port1 },
          { Name: pair.Port2, IsUSB: false, VID: '', PID: '', SerialNumber: '', FriendlyName: pair.Port2 },
        );
        return pair;
      },
      4156784338: (id: string, portName: string) => {
        if (mockState.virtualPorts.some(port => port.ID === id)) {
          throw new Error('virtual port ID already exists');
        }
        const vport = { ID: id, Port: `/tmp/${portName}` };
        mockState.virtualPorts.push(vport);
        mockState.ports.push({
          Name: vport.Port,
          IsUSB: false,
          VID: '',
          PID: '',
          SerialNumber: '',
          FriendlyName: vport.Port,
        });
        return vport;
      },
      1660406268: (id: string) => {
        mockState.virtualPairs = mockState.virtualPairs.filter(pair => pair.ID !== id);
        return null;
      },
      2248938995: (id: string) => {
        mockState.virtualPorts = mockState.virtualPorts.filter(port => port.ID !== id);
        return null;
      },
      3694945770: () => mockState.virtualPairs,
      1806471203: () => mockState.virtualPorts,
      2000913547: (id: string, p1: string, p2: string, baud: number) => {
        if (p1 === p2) {
          throw new Error('cannot bridge a port to itself');
        }
        if (mockState.bridges.some(bridge => bridge.ID === id)) {
          throw new Error('bridge ID already exists');
        }
        const bridge = { ID: id, Port1: p1, Port2: p2, BaudRate: baud };
        mockState.bridges.push(bridge);
        return bridge;
      },
      2952555164: (id: string) => {
        mockState.bridges = mockState.bridges.filter(bridge => bridge.ID !== id);
        return null;
      },
      2577893816: () => mockState.bridges,
      1844679930: (req: any) => ({
        ID: req.ID,
        Status: 'running',
        Error: '',
        Nodes: (req.Nodes ?? []).map((node: any) => ({
          ID: node.ID,
          Type: node.Type,
          Status: 'running',
          RxBytes: node.Type === 'serial.receiver' ? 5 : 0,
          TxBytes: node.Type === 'serial.sender' ? 5 : 0,
          FrameCount: 0,
          ResourceID: '',
          Error: '',
        })),
      }),
      3195045016: (id: string) => ({
        ID: id,
        Status: 'running',
        Error: '',
        Nodes: [],
      }),
      2736010660: () => ({
        Offset: 0,
        Data: btoa(mockState.graphBufferText),
        Total: mockState.graphBufferText.length,
        EOF: true,
      }),
      514153512: () => ({ Frames: [], Total: 0, NextOffset: 0 }),
      2969054863: () => null,
      151526110: () => null,
      3331981752: (req: any) => utf8Length(req.Content ?? ''),
      3380062301: () => null,
      1653301220: () => ({ CPUPercent: 7.5, MemoryBytes: 64 * 1024 * 1024 }),
    };

    const originalFetch = window.fetch.bind(window);
    window.fetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
      const url = typeof input === 'string' ? input : input.toString();
      if (!url.includes('/wails/runtime')) {
        return originalFetch(input, init);
      }

      const body = init?.body ? JSON.parse(init.body as string) : {};
      const descriptor = body.args ?? {};
      const args = descriptor.args ?? [];
      const methodID = descriptor.methodID;
      const handler = handlers[methodID];
      if (!handler) {
        return new Response(`unknown mock method ID: ${methodID}`, { status: 500 });
      }
      try {
        return new Response(JSON.stringify(handler(...args)), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      } catch (error: any) {
        return new Response(error?.message ?? 'mock runtime error', { status: 500 });
      }
    };

    const originalDispatch = window.dispatchEvent.bind(window);
    window.dispatchEvent = (event: Event): boolean => {
      if (event instanceof CustomEvent && event.type === 'mocktrue:serial-data') {
        const data = event.detail;
        const handle = mockState.handles.find(h => h.ID === data.PortID);
        if (handle) handle.RxBytes += data.Data.length;
        (window as any)._wails?.dispatchWailsEvent?.({ name: 'serial:data', data });
      }
      return originalDispatch(event);
    };

    (window as any).Wails = (window as any).Wails || {};
    (window as any).__mockState = mockState;

    (window as any)._wails = (window as any)._wails || {};
  });
}

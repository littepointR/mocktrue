import { spawn } from 'node:child_process';

const passthroughArgs = process.argv.slice(2);
if (passthroughArgs[0] === '--') {
  passthroughArgs.shift();
}

const env = {
  ...process.env,
  PORTWEAVE_E2E_MODE: 'wails',
};

function quoteForCmd(value) {
  const arg = String(value);
  if (!/[\s"&|<>^]/.test(arg)) {
    return arg;
  }
  return `"${arg.replace(/(["^])/g, '^$1')}"`;
}

const child = process.platform === 'win32'
  ? spawn(process.env.ComSpec ?? 'cmd.exe', [
      '/d',
      '/s',
      '/c',
      ['node_modules\\.bin\\playwright.cmd', 'test', ...passthroughArgs].map(quoteForCmd).join(' '),
    ], { stdio: 'inherit', env })
  : spawn('node_modules/.bin/playwright', ['test', ...passthroughArgs], { stdio: 'inherit', env });

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});

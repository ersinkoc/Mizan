import { mkdirSync, rmSync } from 'node:fs';
import { spawn } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const webuiRoot = path.resolve(here, '..');
const repoRoot = path.resolve(webuiRoot, '..');
const home = path.join(repoRoot, 'dist', 'e2e-home');
const bind = process.env.MIZAN_E2E_BIND ?? '127.0.0.1:7891';

rmSync(home, { recursive: true, force: true });
mkdirSync(home, { recursive: true });

const go = process.platform === 'win32' ? 'go.exe' : 'go';
const child = spawn(go, ['run', '../cmd/mizan', 'serve', '--bind', bind, '--home', home], {
  cwd: webuiRoot,
  env: { ...process.env, MIZAN_HOME: home },
  stdio: 'inherit'
});

let shuttingDown = false;
const stop = (signal) => {
  shuttingDown = true;
  if (!child.killed) {
    child.kill(signal);
  }
  setTimeout(() => process.exit(0), 5_000).unref();
};

process.once('SIGINT', () => stop('SIGINT'));
process.once('SIGTERM', () => stop('SIGTERM'));
child.on('exit', (code) => {
  process.exit(code ?? (shuttingDown ? 0 : 1));
});

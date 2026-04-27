import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const webuiRoot = path.resolve(here, '..');
const repoRoot = path.resolve(webuiRoot, '..');
const source = path.join(webuiRoot, 'dist');
const target = path.join(repoRoot, 'internal', 'server', 'dist');

if (!existsSync(source)) {
  throw new Error(`WebUI build output does not exist: ${source}`);
}

rmSync(target, { recursive: true, force: true });
mkdirSync(path.dirname(target), { recursive: true });
cpSync(source, target, { recursive: true });

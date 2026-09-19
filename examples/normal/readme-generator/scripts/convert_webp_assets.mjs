#!/usr/bin/env node
/**
 * Convert README visual assets to WebP using cwebp.
 *
 * WebP keeps README image loading lightweight. This script intentionally uses
 * the local cwebp binary instead of adding a heavy image dependency.
 */

import { execFileSync } from 'node:child_process';
import { existsSync, statSync } from 'node:fs';
import { resolve } from 'node:path';

const DEFAULT_JOBS = [
  ['assets/banner.png', 'assets/banner.webp'],
  ['assets/features.png', 'assets/features.webp'],
];

const args = process.argv.slice(2);
const quality = '86';

function parseJobs() {
  if (!args.length) return DEFAULT_JOBS;
  if (args.length % 2 !== 0) {
    throw new Error('Usage: node scripts/convert_webp_assets.mjs [input.png output.webp]...');
  }

  const jobs = [];
  for (let i = 0; i < args.length; i += 2) {
    jobs.push([args[i], args[i + 1]]);
  }
  return jobs;
}

function ensureCwebp() {
  try {
    execFileSync('cwebp', ['-version'], { stdio: 'ignore' });
  } catch {
    throw new Error('cwebp is required to convert README assets to WebP.');
  }
}

ensureCwebp();

for (const [inputFile, outputFile] of parseJobs()) {
  const input = resolve(inputFile);
  const output = resolve(outputFile);

  if (!existsSync(input) && existsSync(output)) {
    console.log(`${outputFile}: already converted`);
    continue;
  }

  if (!existsSync(input)) {
    throw new Error(`${inputFile} does not exist`);
  }

  execFileSync('cwebp', ['-q', quality, input, '-o', output], { stdio: 'inherit' });

  const before = statSync(input).size;
  const after = statSync(output).size;
  const saved = Math.max(0, before - after);
  console.log(`${outputFile}: ${before} -> ${after} bytes (${saved} saved)`);
}

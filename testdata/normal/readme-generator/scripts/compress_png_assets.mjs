#!/usr/bin/env node
/**
 * Losslessly recompress PNG IDAT chunks with Node's built-in zlib.
 *
 * This is intentionally dependency-free so README asset generation can always
 * run after Playwright screenshots or Codex Image Gen output is copied in.
 */

import { readFileSync, statSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync, inflateSync } from 'node:zlib';

const PNG_SIGNATURE = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
const DEFAULT_FILES = [
  'assets/banner.png',
  'assets/features.png',
];

const files = process.argv.slice(2);
const targets = files.length ? files : DEFAULT_FILES;

const crcTable = new Uint32Array(256);
for (let n = 0; n < 256; n += 1) {
  let c = n;
  for (let k = 0; k < 8; k += 1) {
    c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  }
  crcTable[n] = c >>> 0;
}

function crc32(buffer) {
  let c = 0xffffffff;
  for (const byte of buffer) {
    c = crcTable[(c ^ byte) & 0xff] ^ (c >>> 8);
  }
  return (c ^ 0xffffffff) >>> 0;
}

function makeChunk(type, data) {
  const typeBuffer = Buffer.from(type, 'ascii');
  const length = Buffer.alloc(4);
  const crc = Buffer.alloc(4);
  length.writeUInt32BE(data.length, 0);
  crc.writeUInt32BE(crc32(Buffer.concat([typeBuffer, data])), 0);
  return Buffer.concat([length, typeBuffer, data, crc]);
}

function optimizePng(path) {
  const input = readFileSync(path);
  if (!input.subarray(0, 8).equals(PNG_SIGNATURE)) {
    throw new Error(`${path} is not a PNG file`);
  }

  const passthroughChunks = [];
  const idatChunks = [];
  let offset = 8;

  while (offset < input.length) {
    const length = input.readUInt32BE(offset);
    const type = input.subarray(offset + 4, offset + 8).toString('ascii');
    const data = input.subarray(offset + 8, offset + 8 + length);
    const rawChunk = input.subarray(offset, offset + 12 + length);
    offset += 12 + length;

    if (type === 'IDAT') {
      idatChunks.push(data);
      continue;
    }

    if (type === 'IEND') {
      const inflated = inflateSync(Buffer.concat(idatChunks));
      const recompressed = deflateSync(inflated, { level: 9 });
      const output = Buffer.concat([
        PNG_SIGNATURE,
        ...passthroughChunks,
        makeChunk('IDAT', recompressed),
        rawChunk,
      ]);
      writeFileSync(path, output);
      return;
    }

    passthroughChunks.push(rawChunk);
  }

  throw new Error(`${path} has no IEND chunk`);
}

for (const file of targets) {
  const path = resolve(file);
  const before = statSync(path).size;
  optimizePng(path);
  const after = statSync(path).size;
  const saved = Math.max(0, before - after);
  console.log(`${file}: ${before} -> ${after} bytes (${saved} saved)`);
}

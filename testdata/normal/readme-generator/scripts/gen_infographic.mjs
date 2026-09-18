#!/usr/bin/env node
/**
 * gen_infographic.mjs
 * Playwright screenshot utility for README infographic generation.
 *
 * Usage:
 *   node scripts/gen_infographic.mjs <input.html> <output.png> [width] [height]
 *
 * Examples:
 *   node scripts/gen_infographic.mjs /tmp/banner.html assets/banner.png 1920 1080
 *   node scripts/gen_infographic.mjs /tmp/features.html assets/features.png
 */

import { chromium } from 'playwright';
import { resolve } from 'path';
import { existsSync } from 'fs';

const [,, inputHtml, outputPng, width = '1920', height = '1080'] = process.argv;

if (!inputHtml || !outputPng) {
  console.error('Usage: node gen_infographic.mjs <input.html> <output.png> [width] [height]');
  process.exit(1);
}

const inputPath = resolve(inputHtml);
if (!existsSync(inputPath)) {
  console.error(`Input file not found: ${inputPath}`);
  process.exit(1);
}

const w = parseInt(width, 10);
const h = parseInt(height, 10);

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.setViewportSize({ width: w, height: h });
  await page.goto(`file://${inputPath}`);
  // Wait for fonts (Google Fonts CDN) and any CSS transitions
  await page.waitForTimeout(2500);
  await page.screenshot({
    path: resolve(outputPng),
    clip: { x: 0, y: 0, width: w, height: h },
  });
  await browser.close();
  console.log(`✅ ${outputPng} (${w}×${h})`);
})();

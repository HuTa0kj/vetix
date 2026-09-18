#!/usr/bin/env node
/**
 * Render this skill's own README showcase assets from the HTML templates.
 *
 * The generic screenshot utility stays in gen_infographic.mjs. This script only
 * fills the templates with this repository's current story and calls the utility.
 */

import { execFileSync } from 'node:child_process';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const tmpDir = '/private/tmp/readme-generator-showcase';

mkdirSync(tmpDir, { recursive: true });

const common = {
  PROJECT_NAME: 'GitHub README Generator',
  TAGLINE: '把项目 README 做成一份可以展示的作品。',
  PRIMARY_COLOR: '#c8a15a',
  CATEGORY: 'Portfolio README Design',
  PLATFORM: 'Codex',
  LANGUAGE: 'Markdown',
  VERSION_INFO: 'Story-driven · Image Gen ready',
};

const techCards = `
<div class="tech-card">
  <div class="tech-icon">01</div>
  <div>
    <div class="tech-name">Story</div>
    <div class="tech-desc">先提炼项目为什么存在，而不是先堆功能。</div>
  </div>
</div>
<div class="tech-card">
  <div class="tech-icon">02</div>
  <div>
    <div class="tech-name">Visual</div>
    <div class="tech-desc">封面可用 Codex Image Gen，说明图保持清晰准确。</div>
  </div>
</div>
<div class="tech-card">
  <div class="tech-icon">03</div>
  <div>
    <div class="tech-name">Delivery</div>
    <div class="tech-desc">输出 README、视觉资产和 GitHub 元信息建议。</div>
  </div>
</div>`;

const featureCards = `
<article class="card featured">
  <div>
    <div class="card-tag">Story</div>
    <h3 class="card-title">讲清项目为什么存在</h3>
    <p class="card-desc">把背景、对象和结果压缩成读者能快速判断的叙事。</p>
  </div>
  <div>
    <div class="rule"></div>
    <div class="card-icon">01</div>
  </div>
</article>
<article class="card">
  <div>
    <div class="card-tag">Visual</div>
    <h3 class="card-title">让 README 像作品被看见</h3>
    <p class="card-desc">用封面、海报和留白建立第一印象，避免小字信息堆叠。</p>
  </div>
  <div>
    <div class="rule"></div>
    <div class="card-icon">02</div>
  </div>
</article>
<article class="card">
  <div>
    <div class="card-tag">Signal</div>
    <h3 class="card-title">保留真正有用的信息</h3>
    <p class="card-desc">快速开始、可信结果、作者和许可证，其他噪音交给 docs。</p>
  </div>
  <div>
    <div class="rule"></div>
    <div class="card-icon">03</div>
  </div>
</article>`;

const jobs = [
  {
    template: 'banner.html',
    html: `${tmpDir}/banner.html`,
    png: `${tmpDir}/banner.png`,
    webp: `${root}/assets/banner.webp`,
    vars: { ...common, TECH_CARDS: techCards },
  },
  {
    template: 'features.html',
    html: `${tmpDir}/features.html`,
    png: `${tmpDir}/features.png`,
    webp: `${root}/assets/features.webp`,
    vars: { ...common, FEATURE_CARDS: featureCards, FEATURE_COUNT: '3' },
  },
];

for (const job of jobs) {
  let html = readFileSync(`${root}/templates/${job.template}`, 'utf8');
  for (const [key, value] of Object.entries(job.vars)) {
    html = html.replaceAll(`{{${key}}}`, value);
  }
  writeFileSync(job.html, html);
  execFileSync(
    process.execPath,
    [`${root}/scripts/gen_infographic.mjs`, job.html, job.png, '1920', '1080'],
    { stdio: 'inherit' },
  );
}

execFileSync(
  process.execPath,
  [
    `${root}/scripts/convert_webp_assets.mjs`,
    `${tmpDir}/banner.png`,
    `${root}/assets/banner.webp`,
    `${tmpDir}/features.png`,
    `${root}/assets/features.webp`,
  ],
  { stdio: 'inherit' },
);

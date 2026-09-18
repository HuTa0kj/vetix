# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project follows semantic versioning.

## [Unreleased]

### Added

- Added Codex Image Gen-first visual asset rules with compressed image output before README embedding.
- Added dependency-free PNG recompression script and wired it into the showcase asset generation flow.
- Added first-principles README guidance based on high-star open source project patterns.
- Added visual-budget and noise-audit rules so the Skill can choose between portfolio, clean documentation, and structured README modes.
- Added project name-card image guidance so README visuals carry the project name, positioning line, and a few high-signal labels.
- Added WebP conversion for README showcase assets.

### Changed

- Updated HTML-to-PNG fallback templates to match the black, cinematic, high-contrast magazine-cover visual direction.
- Reduced the default README visual set from three images to two: `banner.png` and `features.png`.
- Replaced the separate workflow image with concise README prose.
- Regenerated `assets/banner.png` and `assets/features.png` with Codex Image Gen instead of HTML-to-PNG screenshots.
- Switched README showcase assets from PNG to WebP for lighter loading.

### Removed

- Removed `assets/workflow.png` and `templates/workflow.html` from the default showcase flow.

## [1.1.0] - 2026-06-02

### Added

- Added portfolio-grade README generation guidance focused on project story, visual quality, and reduced noise.
- Added Codex Image Gen / gpt-image-2 guidance for README cover assets, with clear boundaries for when HTML screenshots remain the better choice.
- Added GitHub Description and Topics recommendation workflow with scoring, star ratings, and optional `gh repo edit` guidance.
- Added `npm run showcase` to regenerate this repository's README visual assets from the templates.

### Changed

- Reworked the README from a feature-heavy explanation into a portfolio-style project introduction.
- Redesigned the banner, features, and workflow templates as large-type editorial posters.
- Regenerated `assets/banner.png`, `assets/features.png`, and `assets/workflow.png` using the new visual system.
- Updated the Skill workflow around story-first README design, visual mode selection, and metadata recommendations.

## [1.0.0] - 2026-04-20

### Added

- Initial README generator Skill.
- Added HTML templates for banner, features, and workflow infographics.
- Added Playwright-based screenshot script for generating 16:9 PNG assets.
- Added README structure guidance, MIT license handling, and GitHub metadata suggestions.

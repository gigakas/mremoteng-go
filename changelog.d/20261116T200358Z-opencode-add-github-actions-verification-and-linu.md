---
timestamp: 2026-11-16T20:03:58Z
agent: opencode
files:
  - .github/workflows/ci.yml
  - .github/workflows/package-linux.yml
  - blueprint/phase-4-packaging.md
  - auditory/phase-4.2-ci-20261116-opencode.md
---

Add GitHub Actions verification and Linux packaging workflows

Stage 4.2 of blueprint/phase-4-packaging.md. ci.yml runs scripts/check.sh and scripts/smoke.sh on every pull request and push to main, calling the same scripts as a developer instead of duplicating the build. package-linux.yml builds the release tarball and the .deb on every push to main and keeps them as run artifacts (amd64, 30-day retention); it verifies the package payload (both binaries, desktop entry, icon, -version) before uploading and refuses to publish a tag whose name disagrees with VERSION. Only a v* tag creates a GitHub Release, using the runner's gh CLI with GITHUB_TOKEN so no third-party release action is needed. Actions are pinned by commit SHA, the workflow default permission is contents: read, and only the release job can write. arm64, rpm, Flatpak and installation acceptance stay pending (4.3/4.5); the audit records them.

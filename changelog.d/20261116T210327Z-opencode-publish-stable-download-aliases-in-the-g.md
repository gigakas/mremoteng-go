---
timestamp: 2026-11-16T21:03:27Z
agent: opencode
files:
  - .github/workflows/package-linux.yml
  - README.md
---

Publish stable download aliases in the GitHub release

The release files carry the version in their names, so a releases/latest/download/<name> link broke with every version and the only permanent downloadable installer was the workflow artifact (login required, 30-day retention). The release job now also publishes mremoteng-go-linux-amd64.deb, mremoteng-go-linux-amd64.tar.gz and a checksum generated for that alias name, next to the versioned files, so the latest links stay valid. The README gained a Download table with those permanent links, a curl + apt install example, and the tag procedure (git tag v$(cat VERSION)) that triggers a release, noting that pushes to main only produce run artifacts.

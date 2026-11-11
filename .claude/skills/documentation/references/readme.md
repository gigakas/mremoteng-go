# README

The README is the front door of the repository. A reader should understand what the project is, whether it is for them, and how to run it **without opening any other file**.

## Audience

Write for a new contributor or user who has the required tools installed but knows nothing about this project. Do not assume internal context.

## Structure

| Section | Question it answers | Required? |
|---------|--------------------|-----------|
| **Title + one-liner** | What is this? | Yes |
| **Why / problem** | Why does it exist? | Yes |
| **Requirements** | What do I need installed? | Yes |
| **Install** | How do I get it running? | Yes |
| **Quickstart** | What is the shortest working example? | Yes |
| **Configuration** | What settings/env vars matter? | If applicable |
| **Usage** | How do I use the main features? | Yes |
| **Architecture** | How do the pieces fit? | For non-trivial projects |
| **Testing** | How do I run the tests? | Yes |
| **Contributing** | How do I contribute? | Yes |
| **License** | Can I use it? | Yes |
| **Links** | Where are the detailed docs? | Yes |

## Skeleton

```markdown
# Project name

One sentence that says what it does and for whom.

## Why

The problem this project solves and the alternative it replaces.

## Requirements

- Python 3.11+ / Node 20+ / Go 1.22+
- Docker (optional)

## Install

```bash
git clone <repo>
cd <repo>
<install command>
```

## Quickstart

```bash
<command>
```

Expected output:

```
<short output>
```

## Configuration

| Variable | Default | Purpose |
|----------|---------|---------|
| `DATABASE_URL` | — | Database connection |
| `LOG_LEVEL` | `info` | Log verbosity |

## Usage

### <Main task>

```bash
<command>
```

## Architecture

Short description + link to [docs/architecture.md](docs/architecture.md).

## Testing

```bash
<test command>
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — see [LICENSE](LICENSE).
```

## Quickstart rules

- It must work from a **clean checkout**.
- It must be **copy-pasteable**; no placeholders unless clearly marked.
- It must show **expected output** so the reader knows it worked.
- It must be the **shortest path** to a working result.
- It must not require reading five other files.

## Configuration

Document only settings a user actually needs. For each one: name, default, purpose, and whether it is required. Prefer a table.

Do not document secrets or real credentials. Show placeholders.

## Badges

Badges are optional; add only ones that are maintained (CI, coverage, version). A broken badge is worse than no badge.

## Architecture pointer

Do not put the whole architecture in the README. A short paragraph plus a link to `docs/architecture.md` or an ADR is enough.

## Testing section

Give the **fast** command first, then the full suite:

```bash
pytest -q                 # fast unit tests
pytest                    # full suite
```

If the project uses the four-layer runner:

```bash
./scripts/check.sh          # formatting check → vet → all tests
go test ./integration/ -count=1
```

## Checklist

- [ ] One-sentence description at the top.
- [ ] Requirements with versions.
- [ ] Install command tested from a clean checkout.
- [ ] Quickstart tested and copy-pasteable.
- [ ] Expected output shown.
- [ ] Configuration table with defaults.
- [ ] Testing commands.
- [ ] Link to architecture/ADRs.
- [ ] Contributing and license.
- [ ] No orphan links.
- [ ] No secrets or real credentials.

## Anti-patterns

| Anti-pattern | Problem | Fix |
|---|---|---|
| "See the wiki" | No front door | Put install/run/test in the README |
| Wall of text with no headings | Unscannable | Use the sections above |
| Placeholder commands | Reader cannot run it | Use real examples |
| Outdated screenshots | Misleading | Regenerate or remove |
| Documenting every internal detail | Noise | Link to detailed docs |
| No testing section | Contributors guess | Show the exact command |

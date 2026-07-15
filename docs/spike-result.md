# Phase 0 result — go/no-go on external-process window embedding

- **Date**: 2026-07-16 · **Prepared by**: claude-code (stage 0.4).
- **Decision owner**: the human. This document presents evidence and
  options; the recommendation is argued, not decided.
- **Decision**: _pending — to be filled in by the human._

## The question Phase 0 had to answer

Can mremoteng-go host RDP (and later AnyDesk) sessions by launching the
official clients as **external processes** and embedding their windows —
without linking libfreerdp (GPLv2 vs Apache-2.0), on both Linux and
Windows?

## Evidence

### Linux — stage 0.1 (`docs/spike-x11.md`)

Validated end-to-end on GNOME (via XWayland): xfreerdp embedded with
`/parent-window`, resize with client-side scaling, focus in/out, process
exit detection. Naive post-hoc reparenting is racy (xfreerdp re-creates
its window) — `/parent-window` is the mechanism.

### Windows — stage 0.2 (`docs/spike-win32.md`)

Validated end-to-end on a Windows 11 VM: `sdl-freerdp` (the current
FreeRDP Windows client) embedded via `SetParent` after solving the DPI
gauntlet (per-monitor-v2 + MIXED hosting before window creation +
post-call verification) with a find-and-adopt retry loop. Notepad control
confirms the mechanism generally; mstsc is unreliable (uiAccess/UIPI) and
demoted to documented fallback.

### Wayland — stage 0.3 (`docs/spike-wayland.md`)

XWayland is the only path (no native-Wayland embedding protocol exists,
July 2026) and it works on GNOME. Hard constraint: the Linux app must run
as an X11 client (no `wayland` Fyne build tag).

## Known gaps (accepted risks if "go")

1. **KDE Plasma Wayland untested** — expectation is parity via XWayland;
   verify when an environment exists, at latest via preview feedback
   (5.3/5.4).
2. **`/dynamic-resolution` untested against a real Windows RDP server** —
   fails against the xrdp container on both platforms; `smart-sizing`
   (blurrier at non-native sizes) is the universal fallback today.
3. **FreeRDP Windows binaries are nightly-only** — packaging (4.4) must
   pin/mirror or self-build; no license issue (external process only).
4. **mstsc cannot be relied on** as the Windows client — FreeRDP is the
   client; mstsc stays a degraded fallback.

## Options

- **A. Go** — proceed with the master plan: external process + embedding
  for RDP (2.5) and AnyDesk (2.7), native Go for SSH/Telnet/VNC/etc. The
  premise held on both platforms with the real client; the gaps above are
  scoped and owned by later stages.
- **B. Conditional go** — proceed, but gate 2.5 on first validating KDE
  Wayland and dynamic resolution against a real Windows RDP host (delays
  protocol work; buys certainty on the two visual-quality gaps).
- **C. Rethink** — treat embedding as too fragile and explore
  alternatives before Phase 2 (e.g. RDP protocol library in Go — none
  mature exists; or shipping without integrated RDP — parity loss).
  Nothing in the evidence supports this today.

## Recommendation

**Option A (go).** Both platforms validated with the production client;
every failure encountered had an identifiable cause and a documented
recipe; the remaining gaps are quality trade-offs, not viability risks,
and each has an owner in the plan. Option B's gates would serialize work
that can proceed in parallel (the gaps don't block Phase 1 or stages
2.1–2.4 at all).

## On "go": immediate consequences

- Phase 0 closes; spike code (`internal/spike/`) is deleted per charter —
  findings live in `docs/spike-*.md`.
- Top-level `README.md` updated with the outcome.
- Phase 1 proceeds without reservation (1.3 already done); 2.5/2.7 inherit
  the field manuals.

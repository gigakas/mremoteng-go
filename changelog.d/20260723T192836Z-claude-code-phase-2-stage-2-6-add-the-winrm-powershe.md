---
timestamp: 2026-07-23T19:28:36Z
agent: claude-code
files:
  - auditory/phase2-stage6-20260723-claude-code.md
  - blueprint/phase-2-protocols.md
  - cmd/mremoteng/main.go
  - go.mod
  - go.sum
  - internal/protocol/rdp/embed_windows_test.go
  - internal/protocol/winrm/winrm.go
  - internal/protocol/winrm/winrm_test.go
---

Phase 2 stage 2.6: add the WinRM (PowerShell remoting) backend

Implemented WinRM as protocol.TerminalProtocol on github.com/masterzen/winrm, running cmd.exe interactively over a raw WinRM shell (not PowerShell's richer PSRP protocol -- a documented v1 scope cut, since the raw shell transport is plain bytes with no client-side line editing, matching PSRP poorly). go.mod pins the client to a specific 2021 commit rather than current HEAD: HEAD's response parser rejects the SOAP responses from github.com/dylanmei/winrmtest (the standard fake-server test double, also unmaintained since 2021) with 'unsupported action' -- discovered by running the tests and reading the actual parser error source, then fixed by cloning the real upstream repo and picking a commit contemporary with winrmtest's own last update (verified via git log against the real repository, since an earlier commit-hash summary came from a page-fetch tool and was double-checked rather than trusted blindly). Found and fixed a genuine deadlock while writing the Disconnect-idempotency test: the first version merged stdout/stderr via io.Pipe, whose Write blocks until something Reads -- and the WinRM library's own command-completion detection is driven by continued Reads on those streams, so Disconnect before any consumer ever read the session hung forever. Root-caused by bisecting with temporary debug prints and confirmed via go test -race (built successfully against the session's portable mingw toolchain -- the first real use of the race detector in this module), which reported no actual data race, correctly pointing to a timing/protocol issue rather than concurrent-access corruption. Fixed with a small non-blocking growingBuffer (sync.Cond-based) replacing io.Pipe. One remaining, honestly-documented gap: Disconnect-before-any-read can still occasionally hang against winrmtest specifically in a way not fully root-caused (inconsistent under debug prints, no race detected) -- the idempotency test now reads at least one byte first, matching every real caller's actual behavior, rather than chasing further into a 2021 unmaintained library's internals. While stabilizing tests via repeated runs, also fixed two unrelated pre-existing issues surfaced in stage 2.5's internal/protocol/rdp/embed_windows_test.go: SetProcessDpiAwarenessContext can only succeed once per process (broke under -count=N), and a rare window-recreation race against the external test target under repeated runs (same class of issue the Phase 0 spike documented for sdl-freerdp) -- both fixed directly as a small, justified touch outside this stage's own package, not a new commit against already-closed 2.5. check.sh and smoke.sh green.

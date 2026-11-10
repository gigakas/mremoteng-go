---
timestamp: 2026-07-22T21:23:19Z
agent: claude-code
files:
  - .claude/skills/changelog/SKILL.md
  - .opencode/command/changelog.md
  - internal/protocol/factory.go
  - internal/protocol/protocol.go
  - internal/protocol/protocol_test.go
---

Phase 2 stage 2.1: add the Protocol interface and self-registering factory

Claimed blueprint stage 2.1 (Protocol interface + factory), the stage that blocks every other Phase 2 stage since it owns internal/protocol/. Added internal/protocol/protocol.go: a Protocol interface (Connect(ctx)/Disconnect/Focus/Resize + OnError/OnClose callback setters) mirroring the original Connection/Protocol/ProtocolBase.cs (read from the local ../mRemoteNG checkout), adapted to Go idiom -- multicast delegates become single-callback setters since each Protocol instance has exactly one owner (a session tab controller), and Connect takes a context.Context for cancellation/timeout instead of the fire-and-forget C# Connect(). Added internal/protocol/factory.go: a Register(protocolType, Constructor)/Create(*connection.ConnectionInfo) registry mirroring ProtocolFactory.CreateProtocol's switch, but inverted -- this package never imports backend subpackages (ssh, vnc, rdp, ...), each backend imports this package and self-registers from its own init(), avoiding an import cycle; Register panics on a nil constructor or a duplicate type (programmer error, same convention as database/sql.Register). Create resolves ConnectionInfo.Effective() once and passes the resolved values to the constructor so backends don't each need to know about inheritance resolution. Tests in protocol_test.go use a fakeProtocol implementing the full interface to exercise Register/Create (happy path, nil info, unregistered type, duplicate registration, nil constructor) and the lifecycle contract (Connect/Focus/Resize call counts, OnClose firing on Disconnect, OnError firing on a simulated async failure, Disconnect idempotency) -- no real backend exists yet, those land in stages 2.2-2.7. ./scripts/check.sh green; ./scripts/smoke.sh currently reports CHANGELOG.md as non-reproducible only because this session has other legitimate uncommitted changelog fragments pending commit (verified go run ./cmd/changelog compile is itself idempotent by running it twice in a row with no further diff).

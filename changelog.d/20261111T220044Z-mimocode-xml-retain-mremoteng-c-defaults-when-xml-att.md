---
timestamp: 2026-11-11T22:00:44Z
agent: mimocode-xml
files:
  - internal/connection/model.go
  - internal/serialize/xml/attributes.go
  - internal/serialize/xml/deserialize_test.go
  - internal/serialize/xml/node26.go
  - internal/serialize/xml/node27.go
---

Retain mRemoteNG C# defaults when XML attributes are missing

Deserialization assigned every connection attribute unconditionally, so a node
whose XML omits an attribute collapsed to the Go zero value and lost the value
mRemoteNG ships. Only Port was rescued later by internal/transfer, every other
field was silently lost on import: UseCredSsp and AutomaticResize became false,
Colors became an empty string that is not a valid enum member, RenderingEngine,
Icon, Panel, PuttySession and EC2Region became empty, RedirectSound DoNotPlay
became BringToThisComputer, VNCEncoding EncHextile became EncRaw,
VNCSmartSizeMode SmartSAspect became SmartSNo, VNCCompression CompNone became
Comp0, RDGatewayUseConnectionCredentials Yes became No and RDPVersion Rdc10
became Highest on 2.7/2.8.

Root cause: two contradictory default tables. internal/connection held the real
C# defaults inline in NewConnectionInfoWithID, while
internal/serialize/xml/attributes.go carried a second, different set inside its
enum helpers.

- internal/connection: the inline literal table is extracted into the exported
  Defaults() ConnectionValues, and NewConnectionInfoWithID calls it (no
  behaviour change). Defaults() is now the single source of truth.
- internal/serialize/xml/attributes.go: new stringDefault, booleanDefault and
  integerDefault accessors are default-aware - they return the supplied default
  only when the key is absent, so a present but malformed value still goes
  through the existing strict parser path. The enum helpers for RDPVersion,
  RenderingEngine, RDGatewayUseConnectionCredentials, Colors, RedirectSound,
  VNCCompression, VNCEncoding and VNCSmartSizeMode now fall back to
  connection.Defaults() instead of their local table.
- node26.go: the Protocol read now happens before Port, which uses
  integerDefault with connection.DefaultPort(protocol). Name, PuttySession, Icon
  and Panel use stringDefault; UseCredSsp and AutomaticResize use booleanDefault
  with true. node27.go: EC2Region uses stringDefault.
- deserialize_test.go: the misleadingly named UseCSharpDefaults test is renamed
  to TestDeserialize_MalformedAttributes_UseDocumentedFallbacks and pins the
  malformed-value path (Port 0, UseCredSsp false, Colors16Bit, CompNone). The new
  TestDeserialize_AbsentAttributes_RetainCSharpDefaults deserializes a node with
  attributes omitted for 2.6, 2.7 and 2.8 and asserts the real C# defaults: Port
  3389, UseCredSsp and AutomaticResize true, Colors16Bit, Rdc10, EdgeChromium,
  DoNotPlay, EncHextile, SmartSAspect, Yes, plus the string defaults. The corpus
  never covers this case because every C# fixture writes those attributes.

Open questions, deliberately left unchanged because the repository has no
evidence for their C# value: RedirectClipboard, VNCProxyPort,
RDPMinutesToIdleTimeout, CacheBitmaps, DisplayWallpaper and DisplayThemes still
default to the Go zero value when absent. Confirming them needs a C# fixture
that omits the attributes or the upstream property initialisers.

Validation: go build ./internal/connection/... ./internal/serialize/...;
go test ./internal/serialize/xml/ ./internal/connection/ -count=1;
go test ./integration/ -run '^TestCSharpCorpus_' -count=1 (unchanged, the corpus
round trip still matches every C# fixture). The full ./scripts/check.sh gate is
left to the lead because another session is editing internal/ui and go.mod.

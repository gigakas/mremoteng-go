---
timestamp: 2026-11-12T01:38:26Z
agent: mimocode
files:
  - internal/connection/model.go
---

Extract the connection default table into connection.Defaults()

The default values for a connection lived as an inline literal inside NewConnectionInfoWithID, which made them unreachable for any other consumer and left the XML deserializer with a second, contradictory table of its own. Extracted the literal into an exported Defaults() function that the constructor now calls, so there is a single place that defines what a fresh connection looks like. This is a pure extraction: the values are identical, the constructor behaves the same and no caller changed.

---
timestamp: 2026-11-12T02:16:15Z
agent: mimocode
files:
  - internal/ui/properties.go
---

Resolve the welcome reference icons when the page is built

The icon samples were package-level variables calling theme.MenuIcon and friends, and those accessors read the current Fyne application. At package initialization no application exists yet, so every entry logged 'Attempt to access current Fyne app when none is started' to the process log: the version smoke run printed the error before it printed the version, and the window startup wrote it repeatedly. The rows now hold the accessor rather than the resource and call it while building the page, which is after the application is up. Verified with a gate and a smoke run that print no Fyne errors.

# internal/spike — Phase 0 throwaway prototypes

Code here validates the "external process + window reparenting" approach
(see `blueprint/phase-0-spike.md`). It is **deleted when Phase 0 closes**;
findings live on in `docs/spike-*.md`. Being throwaway, it is exempt from
the unit-test rule — validation is manual against the checklist below.

## Stage 0.1 — X11 reparenting (`x11reparent/`)

Test host (xrdp in a container, credentials `abc`/`abc`):

```bash
podman run -d --name spike-xrdp -p 3389:3389 \
  -e PUID=1000 -e PGID=1000 lscr.io/linuxserver/rdesktop:ubuntu-xfce
```

Run the spike (needs `xfreerdp` and Fyne build deps: `gcc`,
`libgl1-mesa-dev`, `xorg-dev`, `libxkbcommon-dev`):

```bash
go run -tags spike ./internal/spike/x11reparent -host 127.0.0.1:3389 -user abc -pass abc
```

Spike code is guarded by the `spike` build tag so `./scripts/check.sh`
stays green for agents without the C build dependencies.

### Validation checklist (stage exit)

- [ ] Click Connect: the RDP session appears inside the Fyne window.
- [ ] Resize the Fyne window: the embedded session follows.
- [ ] Keyboard focus enters the session (typing lands in the remote
      desktop) and leaves it (toolbar buttons still clickable).
- [ ] Kill the container (`podman stop spike-xrdp`) or the process: the
      panel detects the exit and reports cleanup.
- [ ] Disconnect button kills the session cleanly.

Record results and anomalies in `docs/spike-x11.md`.

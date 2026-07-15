# Spike 0.3 findings — Wayland assessment

- **Date**: 2026-07-15 · **Agent**: claude-code.
- **Environment actually tested**: Ubuntu, GNOME Shell 50.1, native
  **Wayland** session (`XDG_SESSION_TYPE=wayland`), FreeRDP 3.24.2.
- **KDE: not tested** — no Plasma install available on this machine. See
  pending action below.

## What works (via XWayland)

The entire stage 0.1 validation ran on this Wayland session, transparently
through XWayland (see `docs/spike-x11.md`): embedding with
`/parent-window`, resize with client-side scaling, keyboard focus in/out,
process-exit cleanup. **No compositor-specific glitches observed on GNOME.**

Two conditions make this work:

1. **The Fyne app must run as an X11 client** (default Fyne build, no
   `wayland` build tag). Built with the `wayland` tag, the window would be
   a native Wayland surface, `driver.X11WindowContext` is unavailable and
   there is no parent window id to embed into. The production app must
   either build X11-only or force the X11 backend when embedding is needed.
2. **xfreerdp is also an X11 client**, so parent and child live in the same
   XWayland server and X11 reparenting semantics fully apply.

## What degrades

- **Fractional scaling**: XWayland surfaces are scaled by the compositor;
  on non-integer factors the session can render slightly blurry compared
  with a native Wayland surface. Cosmetic, not functional.
- **HiDPI coordinate math** is on us (physical px vs Fyne logical points —
  already handled in the spike with `Canvas().Scale()`).

## Native-Wayland embedding: state of the art (2026-07)

- **`xdg-foreign`** (the protocol xfreerdp would need): it only lets a
  client set a *parent-child relationship* between toplevels (dialog
  positioning/modality). It does **not** provide visual embedding of a
  foreign surface inside another client's window. No embedding path here.
- **xfreerdp** has no xdg-foreign integration; `xfreerdp3` is X11-only by
  design ([FreeRDP discussion #11595](https://github.com/FreeRDP/FreeRDP/discussions/11595)).
- **wlfreerdp** (the old native Wayland client) was **deprecated with
  FreeRDP 3.0** ([Debian wlfreerdp3 manpage](https://manpages.debian.org/unstable/freerdp3-wayland/wlfreerdp3.1.en.html));
  its replacement is the SDL client (`sdl-freerdp`), which renders natively
  on Wayland ([FreeRDP 3.2 Wayland fixes](https://www.phoronix.com/news/FreeRDP-3.2-Released))
  but creates its own toplevel — there is no mechanism to embed it into our
  window.

**Conclusion**: as of July 2026 there is no viable native-Wayland embedding
mechanism for external processes. XWayland is the only path, and it works.

## Risk and recommendation

- XWayland remains a first-class component of GNOME and KDE today; distros
  ship xfreerdp as the standard RDP client. Risk of XWayland removal in the
  app's lifetime: low, but it is the single point of failure for embedded
  sessions on Wayland — revisit at Phase 5 (cutover) with fresh data.
- **Recommendation for 0.4: go**, with the hard constraint "the app runs
  as an X11/XWayland client on Linux".

## Pending

- Repeat the 0.1 checklist on KDE Plasma Wayland when an environment is
  available (VM or preview-channel user feedback in stage 5.3/5.4). KWin's
  XWayland handling differs from mutter's; expectation is parity, but it is
  unverified.

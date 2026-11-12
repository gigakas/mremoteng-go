---
timestamp: 2026-11-12T02:16:25Z
agent: mimocode
---

Shrink the welcome reference icons to 16 pixels

The icon samples in the button reference were drawn in a 28 pixel box, which is closer to a toolbar control than to a reminder, and they dominated the text next to them. The box is now 16 pixels, defined as welcomeIconSize so the list stays consistent, and Fyne scales the resource down to fit. This rode inside the previous commit instead of its own; the fragment is added separately so the change is recorded.

---
timestamp: 2026-11-14T17:38:27Z
agent: mimocode
files:
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
---

Tell folders and files apart with a coloured icon in each row

The rows already carried an icon, but both samples were drawn in the foreground colour at 18 pixels, so a directory and a file looked the same until you read the trailing slash. A directory now uses the folder sample tinted with the accent colour and a file the document sample in the foreground colour, both at 20 pixels in a slightly taller row. The two resources are resolved per row rather than as package variables on purpose: the theme accessors need a running Fyne application, and a package-level call logs an error before one exists, which is a mistake already made once in this repository. Added TestFileRow_Icons_DistinguishFoldersFromFiles, which fails if both kinds share an icon or if either stops being a folder or document sample. Verified through the rendered widget tree that a directory reports primary_folder.svg and a file foreground_file.svg. Validated with ./scripts/check.sh, whose output contains no Fyne errors.

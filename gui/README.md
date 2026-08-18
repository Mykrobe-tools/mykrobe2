# GUI

This directory contains the Godot 4.7 desktop GUI for `mykrobe2`.

## Development

The project is designed to work in two modes:

1. In the editor / local development
   - it prefers `MYKROBE2_BINARY` if set
   - otherwise it installs a local copy from `../build/mykrobe2`

2. In an exported app
   - it installs a local copy from the bundled `res://bin/mykrobe2` or `res://bin/mykrobe2.exe`

At runtime the GUI executes the installed copy from its custom Godot user data directory:
- macOS: `~/Library/Application Support/mykrobe2/bin/mykrobe2`
- Linux: `~/.local/share/mykrobe2/bin/mykrobe2`
- Windows: `%APPDATA%\mykrobe2\bin\mykrobe2.exe`

This matches the `seqhiker` pattern: bundled binary in the app, installed executable copy in Godot user data.

## Current scope

The current GUI supports:

- choosing reads input
- choosing installed panel data directory
- entering species and panel
- toggling common predict flags
- dragging a reads file or existing JSON result onto the window
- running `mykrobe2 predict`
- viewing split result sections:
  - Overview
  - Drugs
  - Species
  - Evidence
  - Raw JSON

It is intentionally scene-driven: layout and controls live in `scenes/main.tscn`, with logic kept in scripts.

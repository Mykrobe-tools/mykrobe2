# GUI

This directory contains the Godot 4.4 desktop GUI for `mykrobe2`.

## Development

The project is designed to work in two modes:

1. In the editor / local development
   - it prefers `MYKROBE2_BINARY` if set
   - otherwise it falls back to `../build/mykrobe2` relative to this repo

2. In an exported app
   - it expects the bundled CLI binary at:
     - `bin/darwin-amd64/mykrobe2`
     - `bin/darwin-arm64/mykrobe2`
     - `bin/linux-amd64/mykrobe2`
     - `bin/linux-arm64/mykrobe2`
     - `bin/windows-amd64/mykrobe2.exe`
     - `bin/windows-arm64/mykrobe2.exe`

The export pipeline can later copy the correct binary into the matching app bundle.

## Current scope

The first GUI screen supports:

- choosing reads input
- choosing installed panel data directory
- entering species and panel
- toggling common predict flags
- running `mykrobe2 predict`
- viewing a short summary plus raw JSON output

It is intentionally scene-driven: layout and controls live in `scenes/main.tscn`, with logic kept in `scripts/main.gd`.

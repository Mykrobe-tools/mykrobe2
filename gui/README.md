# GUI

This directory contains the Godot 4.7 desktop GUI for `mykrobe2`.

## Local prerequisites

Running the project only needs Godot 4.7.x and a locally built backend (`../build.sh`).
Install the matching Godot export templates as well if you want to create packaged
apps locally. Cross-platform release exports are built by GitHub Actions, so local
Wine, Windows SDKs, and Linux cross-compilers are not required for normal testing.

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
- following the operating system's light/dark appearance, with a saved override
- viewing installed panel versions, references, and descriptions

It is intentionally scene-driven: layout and controls live in `scenes/main.tscn`, with logic kept in scripts.

## Release artifacts

Tags beginning with `v` build GUI applications for Linux and Windows on amd64 and
arm64, plus one universal macOS app that runs natively on Intel and Apple Silicon.
The matching `mykrobe2` command-line executable is bundled inside every GUI app.

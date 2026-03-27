# Bundled CLI binary

This directory holds the canonical bundled backend binary for Godot exports:

- `mykrobe2`
- `mykrobe2.exe` on Windows targets

Build it with the standard Go build script, then copy the matching target binary here:

- `../build.sh`
- `../build.sh --os darwin --arch arm64`
- `../build.sh --os linux --arch amd64`

For a full packaged app export, use:

- `../build_release.sh --target darwin/arm64 --preset "macOS" --out ../dist/mykrobe2.app`

`build_release.sh` now calls `build.sh` internally and copies the correct target binary into this directory before exporting the Godot app.

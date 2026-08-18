# Bundled CLI binary

This directory holds the canonical bundled backend binary for Godot exports:

- `mykrobe2`
- `mykrobe2.exe` on Windows targets

Build it with the standard Go build script, then copy the matching target binary here:

- `../build.sh`
- `../build.sh --os darwin --arch arm64`
- `../build.sh --os linux --arch amd64`

For a full packaged app export, use:

- `../../build_release.sh --target darwin/universal --preset "macOS Universal" --out dist/mykrobe2-gui.dmg`

`build_release.sh` calls `build.sh` internally and stages the correct target
binary without modifying this directory. For macOS it combines the amd64 and
arm64 Go executables into the universal backend bundled in the app.

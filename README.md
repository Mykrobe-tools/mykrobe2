# Mykrobe2

Experimental Go reimplementation of mykrobe.

This is an alpha release and is not yet recommended for production use.

## Install

1. Go to the [releases page](../../releases).
2. Download the GUI build for your operating system, or a standalone command-line archive:
   - macOS GUI: `mykrobe2-gui-*-macos-universal.dmg`
   - Linux GUI: `mykrobe2-gui-*-linux-{amd64,arm64}.tar.gz`
   - Windows GUI: `mykrobe2-gui-*-windows-{amd64,arm64}.zip`
   - command line: an archive without `-gui-` for your operating system and architecture
3. Open the macOS disk image, or extract the Linux/Windows archive, then run `mykrobe2`.

The macOS alpha build is unsigned. On first launch, Control-click the app, choose
**Open**, then confirm that you want to open it.

## Building

Run `./build.sh` to build the command-line executable for the current machine.
Run `./build.sh --release --version v0.1.0-alpha.1` to create compressed standalone CLI
artifacts for macOS, Linux, and Windows on amd64 and arm64.

The GitHub Actions release workflow runs for `v*` tags. It publishes the six
standalone CLI archives, Linux and Windows GUI apps for both architectures, and
one universal macOS GUI app. See `gui/README.md` for local GUI requirements.

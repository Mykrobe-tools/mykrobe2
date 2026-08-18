# Mykrobe2

Experimental Go reimplementation of mykrobe.

This is an alpha release and is not yet recommended for production use.

It was developed with substantial coding assistance from
[OpenAI Codex](https://openai.com/codex), which helped with implementation,
tests, documentation, and benchmarking under human direction and review.

## Install

### Desktop app

1. Go to the [releases page](../../releases).
2. Download the `mykrobe2-gui` build for your system:
   - macOS: `mykrobe2-gui-*-macos-universal.dmg`
   - Linux: `mykrobe2-gui-*-linux-{amd64,arm64}.tar.gz`
   - Windows: `mykrobe2-gui-*-windows-{amd64,arm64}.zip`
3. Open the macOS disk image, or extract the Linux/Windows archive, then run the app.

The macOS alpha build is unsigned. On first launch, Control-click the app, choose
**Open**, then confirm that you want to open it.

### Command-line program

1. Go to the [releases page](../../releases).
2. Download the `mykrobe2` archive without `-gui-` for your operating system and architecture.
3. Extract the archive and place `mykrobe2` (or `mykrobe2.exe` on Windows) on your `PATH`.

## Building

Run `./build.sh` to build the command-line executable for the current machine.
Run `./build.sh --release --version v0.1.0-alpha.1` to create compressed standalone CLI
artifacts for macOS, Linux, and Windows on amd64 and arm64.

The GitHub Actions release workflow runs for `v*` tags. It publishes the six
standalone CLI archives, Linux and Windows GUI apps for both architectures, and
one universal macOS GUI app. See `gui/README.md` for local GUI requirements.

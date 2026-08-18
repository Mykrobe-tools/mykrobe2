# Mykrobe2

Experimental Go reimplementation of mykrobe.

Not recommended for public use yet.

## Building

Run `./build.sh` to build the command-line executable for the current machine.
Run `./build.sh --release --version v0.1.0` to create compressed standalone CLI
artifacts for macOS, Linux, and Windows on amd64 and arm64.

The GitHub Actions release workflow runs for `v*` tags. It publishes the six
standalone CLI archives, Linux and Windows GUI apps for both architectures, and
one universal macOS GUI app. See `gui/README.md` for local GUI requirements.

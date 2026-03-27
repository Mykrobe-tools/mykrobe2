# Bundled CLI binary

This directory holds the canonical bundled backend binary for Godot exports:

- `mykrobe2`
- `mykrobe2.exe` on Windows targets

Build it with:

- `../build_mykrobe2_bins.sh`
- `../build_mykrobe2_bins.sh --target darwin/arm64`
- `../build_mykrobe2_bins.sh --target linux/amd64`

Matrix artifacts for release automation are written to:

- `bin/targets/mykrobe2_<os>_<arch>[.exe]`

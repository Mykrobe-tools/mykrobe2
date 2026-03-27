# Bundled CLI binaries

Exported GUI apps are expected to include the matching `mykrobe2` CLI binary under:

- `bin/darwin-amd64/mykrobe2`
- `bin/darwin-arm64/mykrobe2`
- `bin/linux-amd64/mykrobe2`
- `bin/linux-arm64/mykrobe2`
- `bin/windows-amd64/mykrobe2.exe`
- `bin/windows-arm64/mykrobe2.exe`

The Godot GUI resolves the binary from this layout at runtime outside the editor.

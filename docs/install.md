# Install

## Desktop app

1. Go to the [releases page](https://github.com/Mykrobe-tools/mykrobe2/releases).
2. Download the `mykrobe2-gui` build for your system:
   - macOS: `mykrobe2-gui-*-macos-universal.dmg`
   - Linux: `mykrobe2-gui-*-linux-{amd64,arm64}.tar.gz`
   - Windows: `mykrobe2-gui-*-windows-{amd64,arm64}.zip`
3. Open the macOS disk image, or extract the Linux/Windows archive, then run the app.
4. When you first run the app, it will download and index all the panel data. This will take several minutes.


### macOS notes

The macOS app is unsigned, so macOS will block it from running after you download it.
You may not be able to allow it via Privacy & Security in System Settings. The following commands remove the quarantine attribute directly

If you have an administrator account on your Mac: copy the mykrobe2.app into
Applications, as is normal for apps. Then run this in a terminal:

```bash
sudo xattr -dr com.apple.quarantine /Applications/mykrobe2.app
```

If you do not have an administrator account: create an Applications folder in your home directory and copy mykrobe2.app into it. Then run:

```bash
xattr -dr com.apple.quarantine ~/Applications/mykrobe2.app
```

## Command-line program

1. Go to the [releases page](https://github.com/Mykrobe-tools/mykrobe2/releases).
2. Download the `mykrobe2` archive without `-gui-` for your operating system and architecture.
   - macOS: `mykrobe2-vX.Y.Z-darwin-{amd64,arm64}.tar.gz`
   - Linux: `mykrobe2-vX.Y.Z-linux-{amd64,arm64}.tar.gz`
   - Windows: `mykrobe2-vX.Y.Z-windows-{amd64,arm64}.zip`
3. Extract the archive and place `mykrobe2` (or `mykrobe2.exe` on Windows) on your `PATH`.

On macOS, remove the download quarantine attribute from the extracted executable:

```bash
xattr -d com.apple.quarantine mykrobe2
```

After getting the command-line program, run these two commands to download and index all the panel data:

```bash
mykrobe2 panels update-metadata
mykrobe2 panels update-species all
```

The second command will take several minutes to run. Once finished, the built-in panels will be installed and can use them with `mykrobe2 predict`.

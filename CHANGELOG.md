# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Add a saved desktop GUI scale setting from 75% to 250%.
- Add a compact settings action that downloads the Mykrobe test reads and runs them with the installed default TB panel through the normal GUI analysis flow.
- Add an action to the panel-information dialog for checking for and installing updates to all panels.

### Changed
- Expand and reorganize CLI help with workflow-oriented command ordering, command descriptions, flag descriptions, and usage examples.
- Disable the Godot boot splash image so the correctly sized landing window appears directly.
- Size and centre the initial desktop window from the current screen's usable area, then choose the default UI scale so the landing circle occupies approximately two-thirds of the window height when the user has not saved a scale.
- Make desktop window resizing reflow the layout without resizing text and controls; UI scaling remains an explicit setting.
- Keep the results header fixed, move its settings button into the header, and scroll overflowing content independently on every results tab.
- Make the settings drawer scrollable so all settings remain reachable at larger UI scales and shorter window heights.
- Document how admin and non-admin macOS users can remove quarantine attributes from the unsigned desktop app and CLI executable.

### Fixed
- Restore Mykrobe-compatible `predict` short options, including `-s`, `-k`, `-A`, `-e`, `-D`, `-o`, `-S`, and `-O`, while accepting both `--kmer` and the legacy `--k` spelling.
- Replace the GUI's installed backend after an app update when its reported version or bundled-file hash no longer matches.
- Keep selected panel-information rows visibly selected on hover, and highlight hovered unselected rows in both GUI themes.
- Theme scrollbar and slider tracks and grabbers consistently in both light and dark desktop modes.

## [0.1.0-alpha.1] - 2026-08-18

Initial alpha release, before changelog tracking started in this file.

[Unreleased]: https://github.com/Mykrobe-tools/mykrobe2/compare/v0.1.0-alpha.1...HEAD
[0.1.0-alpha.1]: https://github.com/Mykrobe-tools/mykrobe2/releases/tag/v0.1.0-alpha.1

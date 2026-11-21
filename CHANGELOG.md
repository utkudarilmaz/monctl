# Changelog

All notable changes to this project are written here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-11-21

### Added

- The tray on macOS catches the brightness keys and changes the external monitor.
- The system brightness indicator shows when the tray changes the brightness: on macOS by itself, on GNOME through a small shell extension in the Debian package.
- `monctl brightness` with a value hands the change to a running tray, which is much faster and merges quick changes.
- Input names are found without a config: the active input shows the name of this computer, and an input whose `before` command runs `monctl wake` shows the name of the other computer. The names are kept in `~/.local/state/monctl/names.json`, and a config `label` wins.
- Homebrew formula for macOS, with a service that runs the tray at login.

### Changed

- `monctl input` and the tray now check that the monitor stays on the new input, and try again up to three times when it comes back. This takes about 14 seconds.
- A failed `before` command no longer stops the switch. monctl prints a warning and switches anyway.
- `monctl wake` first prints `computer: <name>` with the name of the computer it runs on.
- A brightness change no longer reads the value back, which makes it about twice as fast.

### Fixed

- Two monctl commands at the same time no longer read each other's replies. They now take turns.

## [0.1.0] - 2026-11-21

### Added

- `list` command that shows the external monitors monctl can reach.
- `brightness` command to read, set, raise, and lower the brightness.
- `input` command to show the current input and switch to another one.
- `probe` command that prints what a monitor reports, to help add new models.
- JSON output for `list`, `brightness`, `input`, and `probe`.
- Monitor profiles, starting with the MSI MPG 274URDFW E16M (DisplayPort and USB-C inputs).
- User config for input labels and hidden inputs.
- Linux support through I2C, with a udev rule for bus access.
- macOS support on Apple Silicon.
- Tray menu for brightness and input on GNOME and in the macOS menu bar.
- Debian packages for amd64 and arm64, and release archives for Linux and macOS.
- `wake` command that turns on the screen of the computer it runs on.
- `before` setting per input in the config: a command monctl runs before it switches to that input, for example to wake the other computer over SSH.
- The tray can run as a systemd user service on Linux and as a launchd agent on macOS.

[Unreleased]: https://github.com/utkudarilmaz/monctl/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/utkudarilmaz/monctl/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/utkudarilmaz/monctl/releases/tag/v0.1.0

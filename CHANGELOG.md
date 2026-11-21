# Changelog

All notable changes to this project are written here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Homebrew formula for macOS, with a service that runs the tray at login.

### Changed

- `monctl input` and the tray now check that the monitor stays on the new input, and try again up to three times when it comes back. This takes about 14 seconds.
- A failed `before` command no longer stops the switch. monctl prints a warning and switches anyway.
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

[Unreleased]: https://github.com/utkudarilmaz/monctl/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/utkudarilmaz/monctl/releases/tag/v0.1.0

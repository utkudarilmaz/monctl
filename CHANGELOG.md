# Changelog

All notable changes to this project are written here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

### Fixed

- The tray icon now shows on GNOME. It was empty before.
- The tray no longer prints an error when the monitor shows another computer.

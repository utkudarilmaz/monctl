# monctl: design

Date: 2026-11-20
Status: approved on 2026-11-20

## Intent

Change monitor settings from the computer instead of the physical buttons on the monitor.

The owner has one monitor shared by two computers: a Debian desktop and a MacBook Pro. They want to change brightness and move the monitor between the two computers without touching it.

The project is open source. It starts with one monitor and should grow to support others.

## Goals

- Raise, lower, and set screen brightness.
- Switch the monitor to another input.
- Work on macOS (Apple Silicon) and Debian with GNOME. Both are required for the first release.
- A command line tool that does everything.
- An optional tray menu on both systems, built on the same core.
- Support the MSI MPG 274URDFW E16M first.
- Make adding another monitor a small data change, not a code change.

## Non-goals for the first release

- Windows.
- Intel Macs.
- Built-in laptop screens.
- Settings other than brightness and input (contrast, volume, color modes).
- Global hotkeys.
- A native GNOME Quick Settings slider or a native Swift app.

## Anti-goals

- Needing `sudo` for daily use.
- Needing a background service for the command line tool to work.
- Sending a command to the wrong display when more than one is attached.
- Writing a value the monitor profile does not list for input switching. A wrong input value can leave the screen dark with no easy way back.

## Constraints

- Monitors take commands over DDC/CI, carried on the video cable. Brightness is VCP code `0x10`. Input source is VCP code `0x60`.
- Linux exposes DDC/CI as `/dev/i2c-*` devices. A normal user cannot open them by default.
- Apple Silicon Macs expose DDC/CI only through the private `IOAVService` API in IOKit. There is no public API. Other open source tools (MonitorControl, m1ddc) use the same calls.
- MSI monitors are known to use non-standard values for input source. The real values must be read from the monitor, not assumed.
- Many monitors answer DDC/CI only on the active input. If that is true here, a computer cannot pull the monitor to itself. It can only push the monitor to the other computer.
- GNOME has no tray by default. Tray icons need the AppIndicator extension, which Debian packages as `gnome-shell-extension-appindicator`.
- The name "MonitorControl" belongs to an existing, well-known macOS app, so the project is named `monctl`.
- On the first target desktop the NVIDIA driver does not link a video output to its I2C bus in sysfs. Detection cannot rely on that link.
- The same desktop has SMBus buses used by the motherboard. Writing to those can damage hardware, so the tool must only open I2C buses that belong to a graphics card.
- The monitor did not answer DDC/CI while its screen was asleep. The tool must report that clearly, not hang or retry forever.

## Approaches

**A. One Go codebase: core library, command line tool, optional tray. Picked.**
One binary per system. On Linux the build is pure Go with no system libraries, which keeps the `.deb` simple. On macOS a small C shim reaches `IOAVService` through cgo. The tray uses `fyne.io/systray`, which talks to the GNOME tray over D-Bus and to the macOS menu bar through Cocoa. The cost: tray menus are plain items, so brightness is steps and presets, not a slider.

**B. The same shape in Rust.**
Just as capable. The Linux tray libraries need GTK and libayatana-appindicator at run time, which makes packaging heavier for no gain here.

**C. Native interface per system over a shared command line tool.**
A Swift menu bar app and a GNOME Shell extension. Best look, with real sliders. The cost: three languages, and GNOME extensions need updates for most GNOME releases.

A was proposed by the builder of this doc and approved by the owner with the build plan. C stays possible later because the command line tool is the stable base.

## Design

### Parts

```
cmd/monctl            command line entry, also starts the tray
internal/ddc          DDC/CI packets: build, checksum, parse, get VCP, set VCP, capabilities
internal/bus          Bus interface and one backend per system
  bus_linux.go        /dev/i2c-* through ioctl
  bus_darwin.go       IOAVService through cgo
  fake.go             in-memory bus for tests
internal/edid         parse maker, product code, serial, name
internal/profile      load and match monitor profiles
internal/monitor      high level actions: brightness, input
internal/tray         tray menu
profiles/             one file per monitor model, embedded in the binary
packaging/            udev rule, .desktop file, LaunchAgent plist
```

`ddc` knows nothing about the operating system. It talks to a `Bus`, which has two operations: write bytes to the display, read bytes from it. This is the seam that makes the protocol testable without a monitor.

### Finding displays

- Linux: list the I2C buses whose parent device is a graphics card. For each connected output in `/sys/class/drm/*/`, use the `ddc` or `i2c-*` link under it when the driver provides one (AMD, Intel). When it does not (NVIDIA), read the EDID at address `0x50` from each graphics bus and match it to the output. Buses that do not belong to a graphics card are never opened. Built-in panels (`eDP`, `LVDS`, `DSI`) are skipped.
- macOS: walk the IORegistry for external displays that have an `IOAVService`, and read the EDID from the same entry.

Each display gets a stable id made from EDID maker, product code, and serial. Commands take `-display <id or index>`. With exactly one supported display attached, the flag is optional. With more than one and no flag, the tool refuses and lists them.

### Monitor profiles

One YAML file per model:

```yaml
match:
  manufacturer: MSI
  productCode: 0x3cf0
name: MSI MPG 274URDFW E16M      # the monitor reports itself as "MPG 274U E16M"
brightness:
  vcp: 0x10
  max: 100
inputs:
  - id: dp1
    label: DisplayPort
    value: 0x0f              # every value must be confirmed on the real monitor
  - id: hdmi1
    label: HDMI 1
    value: 0x11
```

Profiles are embedded in the binary. A user can add or override profiles in `~/.config/monctl/profiles/`.

A display with no profile still gets brightness, using the standard code and the maximum the monitor reports. Input switching on a display with no profile is refused unless the user passes a raw value with `-force`. On a display that has a profile, `-force` is not accepted: only listed inputs can be set.

### User config

Profiles describe a monitor model and are shared by everyone. What is plugged into each port is personal, so it lives in `~/.config/monctl/config.yaml`:

```yaml
displays:
  - match: { manufacturer: MSI, productCode: 0x3cf0
    inputs:
      dp1: { label: Workstation }
      usbc: { label: MacBook }
    hideUnlabeled: true
```

The file is optional. Without it, every input in the profile is shown with its profile label. With it, the command line and the tray show the user's labels, and `monctl input macbook` works as an alias.

The tool does not need to be told which port "this computer" is on. It reads the current input from the monitor, and that is this computer's port. (Builder's call.)

### Command line

```
monctl list
monctl brightness                  show current value
monctl brightness 60 | +10 | -10
monctl input                       list inputs, mark the current one
monctl input hdmi1
monctl probe                       print EDID and capabilities for a new monitor report
monctl tray
```

Exit code is non-zero on any failure. `-json` prints machine readable output with camelCase field names.

### Reliability

DDC/CI is slow and loses messages. The core waits the delay the standard asks for after each command (about 50 ms after a write, 40 ms before reading a reply), retries a failed read up to 3 times, and checks every reply checksum. After setting brightness it reads the value back. After switching input it does not read back, because the monitor may no longer answer.

### Tray

```
[icon]
  Brightness: 60%
  Brighter (+10)
  Dimmer (-10)
  25% / 50% / 75% / 100%
  ───────────
  Input
    * DisplayPort
      HDMI 1
      ...
  ───────────
  Start at login
  Quit
```

- The menu is built from the matched profile.
- State is read from the monitor when the menu opens, not on a timer.
- "Start at login" writes or removes an autostart `.desktop` file on GNOME and a LaunchAgent plist on macOS.
- If the AppIndicator extension is missing on GNOME, `monctl tray` says so and explains how to enable it.

### Permissions

- Debian: the `.deb` installs a udev rule that gives the logged-in user access to I2C devices that belong to graphics outputs (`TAG+="uaccess"`). No group change, no `sudo`.
- macOS: no permission prompt is needed for `IOAVService`.

### Packaging

- GitHub releases built by GoReleaser from a version tag.
- Debian: `.deb` for amd64 and arm64 with the binary, udev rule, and `.desktop` file.
- macOS: Homebrew tap, plus a zipped `.app` for the tray. Without an Apple Developer ID the app is not signed, so the README explains the first-run Gatekeeper step.
- License: MIT.

## Build order

0. **Hardware check, no code.** On the Debian desktop with `ddcutil` and on the MacBook with `m1ddc`: read the EDID and capabilities, confirm brightness, record the input value for each port, and test whether the monitor answers on an inactive input.
   The owner runs this, or gives SSH access so it can be run for them. Steps 1 to 3 do not depend on it and can start first. Step 4 and later wait for its results.
1. Repo base: license, README, contribution guide, Go module, CI on Ubuntu and macOS.
2. `ddc`, `edid`, and the fake bus, built test first.
3. Linux bus and display detection.
4. Profiles and the command line tool. First profile from the step 0 results.
5. macOS bus.
6. Tray on both systems.
7. Packaging and the first release.
8. Later: more profiles, hotkeys, more settings, native interfaces.

## Open, to be settled by the hardware check

- Which monitor port each computer uses.
- The real input source values for this monitor.
- Whether the monitor answers DDC/CI on an inactive input. If not, the tool offers "switch to the other computer" only, and each computer needs the tool installed.
- Whether the Mac reaches DDC/CI on the port it uses. If it does not, the Mac moves to another monitor port or cable until it does. macOS support is required, so this is solved before the macOS backend is written.

## Left to the builder

- Exact retry counts and delays, tuned on the real monitor.
- Choice of YAML library and command line library.
- Tray icon artwork.

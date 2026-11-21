# monctl

Change your monitor's settings from the computer instead of the buttons on the monitor.

`monctl` talks to external monitors over DDC/CI, the control channel that runs on the video cable. It targets Debian with GNOME and macOS with Apple Silicon.

## Status

Early development. The first release is 0.1.0.

Works now on Linux and on macOS with Apple Silicon: `list`, `brightness`, `input`, and `probe`. Intel Macs are not supported.

`monctl tray` works on GNOME with the AppIndicator extension and in the macOS menu bar. It has been started on both, but the menu itself has had little hands on testing so far.

## Usage

```sh
monctl list
```

```text
ID                 NAME           CONNECTOR  BUS         PROFILE
msi-3cf0-ab12cd34  MPG 274U E16M  DP-1       /dev/i2c-5  MSI MPG 274URDFW E16M
```

`list` shows the external monitors monctl can reach. The ID is the maker, the product code, and the serial number from the monitor. On macOS, CONNECTOR shows `-` and BUS shows the display service, for example `avservice:4295167419`.

```sh
monctl brightness        # print the brightness
monctl brightness 40     # set it to 40
monctl brightness +10    # raise it by 10
monctl brightness -10    # lower it by 10
```

Brightness stays between 0 and the monitor's maximum. Each command prints the new value.

```sh
monctl input             # print the current input and the inputs you can switch to
monctl input usbc        # switch to USB-C
```

```text
Current: DisplayPort (dp)
Inputs:
  dp    DisplayPort
  usbc  USB-C
```

```sh
monctl wake              # turn on the screen of this computer
```

`wake` takes no flags. On GNOME it needs this user to be logged in at the screen. On macOS it works like moving the mouse. It is meant to be run from another computer before a switch, see [Configuration](#configuration).

Add `-json` to `list`, `brightness`, `input`, or `probe` for JSON output. With more than one monitor, pick one with `-d`:

```sh
monctl list -json
monctl brightness 40 -d msi-3cf0-ab12cd34
```

`monctl probe` prints what a monitor reports. It helps add a new model, see [CONTRIBUTING.md](CONTRIBUTING.md#adding-a-monitor). Run `monctl` alone for the full command list.

## Configuration

monctl reads `~/.config/monctl/config.json`, or `$XDG_CONFIG_HOME/monctl/config.json` when that variable is set. The path is the same on Linux and macOS. The file is optional.

```json
{
  "inputs": {
    "dp": { "label": "Workstation" },
    "usbc": { "label": "MacBook" },
    "hdmi1": { "hidden": true }
  }
}
```

The keys are the input ids from `monctl input`. A label replaces the name shown, and you can use it as a name: `monctl input MacBook`. A hidden input leaves the list but still works by its id. Ids the monitor does not have are ignored.

`before` is a command monctl runs before it switches the monitor to that input. Write the program and each argument as its own word, because no shell is used. If the command fails or takes longer than 20 seconds, monctl does not switch and prints why. It runs for `monctl input <name>` and for the tray, not for `monctl input -raw`.

Use it to wake the other computer first. Say computers A and B can reach each other over SSH with keys, with no password prompt, and both have monctl installed. On A, give the input that leads to B this setting:

```json
{
  "inputs": {
    "usbc": {
      "label": "Computer B",
      "before": ["ssh", "-o", "BatchMode=yes", "user@computer-b", "monctl", "wake"]
    }
  }
}
```

Do the same on B for the input that leads to A. If monctl is not on the PATH of the other computer, use its full path, for example `/usr/local/bin/monctl`.

Only the config file can hold a `before` command, not a monitor profile. Because it can start programs, make the file writable only by you.

To add or override a monitor profile, put it in `~/.config/monctl/profiles/*.json`. Your profiles win over the ones shipped with monctl.

## Tray menu

The tray is optional. The other commands work without it.

```sh
monctl tray              # show the menu until you pick Quit
monctl tray -d msi-3cf0-ab12cd34
```

```text
Brightness: 60%

Brighter (+10)
Dimmer (-10)
Set brightness   >  0%  25%  50%  75%  100%

Send picture to
  [x] Workstation
  [ ] MacBook

Quit
```

The inputs come from the monitor profile and use your labels from the config. The menu reads the monitor each time you open it. When the monitor shows another computer, the menu says so and its actions are greyed out. Errors go to the terminal that started `monctl tray`.

GNOME shows no tray icons by default. Install the AppIndicator extension, then log out and in:

```sh
sudo apt install gnome-shell-extension-appindicator
gnome-extensions enable ubuntu-appindicators@ubuntu.com
```

`monctl tray` prints these steps when the extension is missing. macOS needs no setup.

### Run the tray as a service

The tray does not start at login by default. As a service, it starts at login and restarts after a crash. Picking Quit stops it until the next login.

On Linux, the Debian package installs a systemd user service. Turn it on:

```sh
systemctl --user enable --now monctl-tray.service
```

It starts with the graphical session. After 5 failed starts in a minute, it gives up. On GNOME it still needs the AppIndicator extension. Without it the service fails, and the status shows the install steps.

```sh
systemctl --user status monctl-tray.service    # see its state
journalctl --user -u monctl-tray.service       # see its messages
systemctl --user disable --now monctl-tray.service   # turn it off
```

For a build from source, copy `packaging/linux/monctl-tray.service` to `~/.config/systemd/user/`. If monctl is not in `/usr/bin`, change the `ExecStart` path in the copy.

On macOS, copy monctl to `/usr/local/bin/`. Then run this from the unpacked archive and log in again:

```sh
cp packaging/io.github.utkudarilmaz.monctl.tray.plist ~/Library/LaunchAgents/
```

To start it now without logging out, and to stop and remove it:

```sh
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/io.github.utkudarilmaz.monctl.tray.plist
launchctl bootout gui/$(id -u)/io.github.utkudarilmaz.monctl.tray
```

## How input switching behaves

- A monitor only answers on the input it is showing. A computer can send the picture to another computer, but it cannot pull it back.
- When the monitor shows another computer, monctl says so.
- The target computer must be awake and sending a picture. If its screen is asleep, the monitor finds no picture and falls back to the old input. With the monitor's automatic input scan turned off, it stays black instead. A laptop with a closed lid may go to sleep when the picture leaves it.
- To fix this, wake the target computer first with a [`before` command](#configuration). In tests, switches with this step worked every time, also onto a blanked screen and onto a laptop with a closed lid.
- Leave the monitor's automatic input scan on, so a failed switch falls back to a picture instead of a black screen.
- After a switch, the monitor needs around ten seconds before it answers again.

## Install

Packages are on the [GitHub releases page](https://github.com/utkudarilmaz/monctl/releases). You can also [build from source](#build-from-source).

### Debian package

Builds exist for amd64 and arm64.

```sh
sudo apt install ./monctl_<version>_amd64.deb
```

The package installs `/usr/bin/monctl` and the udev rule, so you can skip [Linux setup](#linux-setup). It also installs the docs, the licence texts of the libraries monctl uses, and the [tray service](#run-the-tray-as-a-service) in `/usr/lib/systemd/user/`.

### macOS with Homebrew

Apple Silicon only. Homebrew builds monctl on your Mac, so macOS does not block it. You need the Xcode command line tools.

```sh
brew tap utkudarilmaz/monctl https://github.com/utkudarilmaz/monctl
brew install monctl
```

To run the [tray](#tray-menu) at login:

```sh
brew services start monctl
```

### macOS archive

Apple Silicon only. The archive holds the `monctl` binary and a LaunchAgent file for the [tray](#tray-menu).

```sh
tar -xzf monctl_<version>_darwin_arm64.tar.gz
sudo cp monctl_<version>_darwin_arm64/monctl /usr/local/bin/
```

The binary is not signed with an Apple Developer ID, so macOS may block the first start of a downloaded copy. Allow it in System Settings under Privacy and Security. A copy built on the same Mac with `make build` is not affected.

### Build from source

You need Go 1.25 or newer. On macOS, also install the Xcode command line tools, because the macOS part uses cgo with the IOKit framework.

```sh
make build
./bin/monctl version
```

On Linux, then follow [Linux setup](#linux-setup).

## Linux setup

These steps are only for a build from source. The Debian package does them for you.

monctl needs access to the I2C buses of your graphics card. Install the udev rule from this repo:

```sh
sudo cp packaging/linux/60-monctl.rules /etc/udev/rules.d/
sudo udevadm control -R
sudo udevadm trigger -s i2c-dev
```

monctl prints these commands when it cannot open a bus.

The rule gives the logged in user read and write access to every I2C bus on the graphics card, not only the ones that lead to a monitor. monctl itself only opens buses that belong to a PCI graphics device. It never opens SMBus or other system buses.

## macOS setup

No setup is needed. There is no udev rule, no sudo, and no extra permission.

monctl lists only external monitors. The built-in screen is never listed.

monctl uses IOAVService, a private Apple API that other monitor tools for Apple Silicon also use, so a future macOS version could break it.

## Supported monitors

| Monitor | Brightness | Input switching |
|---|---|---|
| MSI MPG 274URDFW E16M | yes | DisplayPort and USB-C. HDMI values are not measured yet. |

On macOS, this monitor is tested over USB-C.

Other monitors are added through small profile files. See [CONTRIBUTING.md](CONTRIBUTING.md#adding-a-monitor).

## Development

```sh
make ci
```

`make ci` runs every check the CI workflow runs: dependency check, lint, tests, build, and `make package`. `make package` writes `dist/monctl_<version>_<os>_<arch>.tar.gz`, and on Linux also `dist/monctl_<version>_<arch>.deb`. The workflow only calls `make` targets, so a failure in CI can be reproduced on your machine with the same command. Run `make` alone to list all targets.

## License

[MIT](LICENSE)

The tray uses [fyne.io/systray](https://github.com/fyne-io/systray) (Apache-2.0) and [godbus/dbus](https://github.com/godbus/dbus) (BSD-2-Clause).

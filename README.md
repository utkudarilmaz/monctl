# monctl

Change your monitor's settings from the computer instead of the buttons on the monitor.

`monctl` talks to external monitors over DDC/CI, the control channel that runs on the video cable. It targets Debian with GNOME and macOS with Apple Silicon.

## Status

Early development. Nothing is released yet.

Works now on Linux and on macOS with Apple Silicon: `list`, `brightness`, `input`, and `probe`. Intel Macs are not supported.

Planned for the first release: an optional tray menu on GNOME and in the macOS menu bar.

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

To add or override a monitor profile, put it in `~/.config/monctl/profiles/*.json`. Your profiles win over the ones shipped with monctl.

## How input switching behaves

- A monitor only answers on the input it is showing. A computer can send the picture to another computer, but it cannot pull it back.
- When the monitor shows another computer, monctl says so.
- The target computer must be awake and sending a picture. If not, the monitor finds no signal and falls back after a few seconds. A laptop with a closed lid may go to sleep when the picture leaves it.
- After a switch, the monitor needs around ten seconds before it answers again.

## Linux setup

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

## Build

You need Go 1.25 or newer. On macOS, also install the Xcode command line tools, because the macOS part uses cgo with the IOKit framework.

```sh
make build
./bin/monctl version
```

## Development

```sh
make ci
```

`make ci` runs every check the CI workflow runs: dependency check, lint, tests, and build. The workflow only calls `make` targets, so a failure in CI can be reproduced on your machine with the same command. Run `make` alone to list all targets.

## License

[MIT](LICENSE)

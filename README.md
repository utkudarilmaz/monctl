# monctl

Change your monitor's settings from the computer instead of the buttons on the monitor.

`monctl` talks to external monitors over DDC/CI, the control channel that runs on the video cable. It targets Debian with GNOME and macOS with Apple Silicon.

## Status

Early development. Nothing is released yet.

Works now:

- `monctl list` on Linux

Planned for the first release:

- Raise, lower, and set screen brightness
- Switch the monitor to another input
- An optional tray menu on GNOME and in the macOS menu bar
- macOS support

## Usage

```sh
monctl list
```

```text
ID                 NAME           CONNECTOR  BUS
msi-3cf0-ab12cd34  MPG 274U E16M  DP-1       /dev/i2c-5
```

`list` shows the external monitors monctl can reach. The ID is the maker, the product code, and the serial number from the monitor. On macOS, `list` reports that the system is not supported yet.

## Linux setup

monctl needs access to the I2C buses of your graphics card. Install the udev rule from this repo:

```sh
sudo cp packaging/linux/60-monctl.rules /etc/udev/rules.d/
sudo udevadm control -R
sudo udevadm trigger -s i2c-dev
```

monctl prints these commands when it cannot open a bus.

The rule gives the logged in user read and write access to every I2C bus on the graphics card, not only the ones that lead to a monitor. monctl itself only opens buses that belong to a PCI graphics device. It never opens SMBus or other system buses.

## Supported monitors

| Monitor | Brightness | Input switching |
|---|---|---|
| MSI MPG 274URDFW E16M | planned | planned |

Other monitors will be added through small profile files. See [CONTRIBUTING.md](CONTRIBUTING.md).

## Build

You need Go 1.25 or newer.

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

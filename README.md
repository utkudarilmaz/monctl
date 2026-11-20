# monctl

Change your monitor's settings from the computer instead of the buttons on the monitor.

`monctl` talks to external monitors over DDC/CI, the control channel that runs on the video cable. It works on Debian with GNOME and on macOS with Apple Silicon.

## Status

Early development. Nothing is released yet.

Planned for the first release:

- Raise, lower, and set screen brightness
- Switch the monitor to another input
- A command line tool
- An optional tray menu on GNOME and in the macOS menu bar

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

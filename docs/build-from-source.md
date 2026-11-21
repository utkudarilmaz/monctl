# Build monctl from source

You need Go 1.25 or newer. On macOS, also install the Xcode command line tools, because the macOS part uses cgo with the IOKit framework.

```sh
make build
./bin/monctl version
```

Copy `bin/monctl` to a folder on your PATH, for example `/usr/local/bin`.

On Linux, then follow [Linux setup](#linux-setup).

## Linux setup

The Debian package does these steps for you. A build from source needs them once.

monctl needs access to the I2C buses of your graphics card. Install the udev rule from this repo:

```sh
sudo cp packaging/linux/60-monctl.rules /etc/udev/rules.d/
sudo udevadm control -R
sudo udevadm trigger -s i2c-dev
```

monctl prints these commands when it cannot open a bus.

The rule gives the logged in user read and write access to every I2C bus on the graphics card, not only the ones that lead to a monitor. monctl itself only opens buses that belong to a PCI graphics device. It never opens SMBus or other system buses.

## Tray service

The steps to run the tray at login from a build are in the [README](../README.md#run-the-tray-as-a-service).

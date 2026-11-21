# Other ways to install monctl

The Debian package and Homebrew are in the [README](../README.md#install). This page covers the macOS archive and a build from source.

## macOS archive

Apple Silicon only. The archive holds the `monctl` binary and a LaunchAgent file for the [tray](../README.md#tray-menu).

```sh
tar -xzf monctl_<version>_darwin_arm64.tar.gz
sudo cp monctl_<version>_darwin_arm64/monctl /usr/local/bin/
```

The binary is not signed with an Apple Developer ID, so macOS may block the first start of a downloaded copy. Allow it in System Settings under Privacy and Security. A copy built on the same Mac with `make build` is not affected.

## Build from source

You need Go 1.25 or newer. On macOS, also install the Xcode command line tools, because the macOS part uses cgo with the IOKit framework.

```sh
make build
./bin/monctl version
```

Copy `bin/monctl` to a folder on your PATH, for example `/usr/local/bin`.

On Linux, then follow [Linux setup](#linux-setup).

### Linux setup

The Debian package does these steps for you. A build from source needs them once.

monctl needs access to the I2C buses of your graphics card. Install the udev rule from this repo:

```sh
sudo cp packaging/linux/60-monctl.rules /etc/udev/rules.d/
sudo udevadm control -R
sudo udevadm trigger -s i2c-dev
```

monctl prints these commands when it cannot open a bus.

The rule gives the logged in user read and write access to every I2C bus on the graphics card, not only the ones that lead to a monitor. monctl itself only opens buses that belong to a PCI graphics device. It never opens SMBus or other system buses.

### Tray service

The steps to run the tray at login from a build are in the [README](../README.md#run-the-tray-as-a-service).

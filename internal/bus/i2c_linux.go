package bus

import (
	"fmt"
	"os"
	"syscall"
)

const ioctlI2CSlave = 0x0703

type i2c struct {
	file *os.File
}

// OpenI2C opens an I2C device such as /dev/i2c-4.
func OpenI2C(path string) (Bus, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return &i2c{file: f}, nil
}

func (b *i2c) Write(addr uint8, data []byte) error {
	if err := b.target(addr); err != nil {
		return err
	}
	n, err := b.file.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("i2c: wrote %d of %d bytes", n, len(data))
	}
	return nil
}

func (b *i2c) Read(addr uint8, n int) ([]byte, error) {
	if err := b.target(addr); err != nil {
		return nil, err
	}
	out := make([]byte, n)
	got, err := b.file.Read(out)
	if err != nil {
		return nil, err
	}
	if got != n {
		return nil, fmt.Errorf("i2c: read %d of %d bytes", got, n)
	}
	return out, nil
}

// DeviceNumber returns the major and minor number of the opened device.
func (b *i2c) DeviceNumber() (major, minor uint32, err error) {
	var st syscall.Stat_t
	if err := syscall.Fstat(int(b.file.Fd()), &st); err != nil {
		return 0, 0, err
	}
	dev := st.Rdev
	major = uint32((dev>>8)&0xfff | (dev>>32)&^0xfff)
	minor = uint32(dev&0xff | (dev>>12)&^0xff)
	return major, minor, nil
}

func (b *i2c) Close() error {
	return b.file.Close()
}

func (b *i2c) target(addr uint8) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, b.file.Fd(), ioctlI2CSlave, uintptr(addr)); errno != 0 {
		return fmt.Errorf("i2c: select address %#x: %w", addr, errno)
	}
	return nil
}

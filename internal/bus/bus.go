// Package bus defines the I2C channel a display is reached through.
package bus

// I2C addresses used by displays.
const (
	AddrDDC  = 0x37
	AddrEDID = 0x50
)

// Bus is one I2C bus that leads to a single display.
type Bus interface {
	Write(addr uint8, data []byte) error
	Read(addr uint8, n int) ([]byte, error)
	Close() error
}

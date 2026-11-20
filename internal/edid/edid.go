// Package edid reads the identity of a display from its EDID base block.
package edid

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
)

// BlockSize is the length of the EDID base block.
const BlockSize = 128

// Errors returned by Parse.
var (
	ErrTooShort    = errors.New("edid: block is shorter than 128 bytes")
	ErrBadHeader   = errors.New("edid: missing header")
	ErrBadChecksum = errors.New("edid: checksum mismatch")
)

var header = []byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}

const (
	descriptorStart = 54
	descriptorSize  = 18
	descriptorCount = 4
	tagName         = 0xfc
	tagSerial       = 0xff
)

// Info identifies a display.
type Info struct {
	Manufacturer string
	ProductCode  uint16
	Name         string
	Serial       string
}

// Parse reads the base block. Extension blocks after the first 128 bytes are ignored.
func Parse(data []byte) (Info, error) {
	if len(data) < BlockSize {
		return Info{}, ErrTooShort
	}
	block := data[:BlockSize]
	if !bytes.Equal(block[:len(header)], header) {
		return Info{}, ErrBadHeader
	}
	var sum byte
	for _, v := range block {
		sum += v
	}
	if sum != 0 {
		return Info{}, ErrBadChecksum
	}

	info := Info{
		Manufacturer: manufacturer(binary.BigEndian.Uint16(block[8:10])),
		ProductCode:  binary.LittleEndian.Uint16(block[10:12]),
	}
	for i := range descriptorCount {
		d := block[descriptorStart+i*descriptorSize:][:descriptorSize]
		if d[0] != 0 || d[1] != 0 || d[2] != 0 {
			continue
		}
		switch d[3] {
		case tagName:
			info.Name = descriptorText(d)
		case tagSerial:
			info.Serial = descriptorText(d)
		}
	}
	if info.Serial == "" {
		if n := binary.LittleEndian.Uint32(block[12:16]); n != 0 {
			info.Serial = strconv.FormatUint(uint64(n), 10)
		}
	}
	return info, nil
}

func manufacturer(v uint16) string {
	letters := make([]byte, 3)
	for i, shift := range []uint{10, 5, 0} {
		n := byte(v >> shift & 0x1f)
		if n < 1 || n > 26 {
			letters[i] = '?'
			continue
		}
		letters[i] = 'A' + n - 1
	}
	return string(letters)
}

func descriptorText(d []byte) string {
	text := d[5:]
	if i := bytes.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	printable := make([]byte, 0, len(text))
	for _, b := range text {
		if b >= 0x20 && b <= 0x7e {
			printable = append(printable, b)
		}
	}
	return strings.TrimSpace(string(printable))
}

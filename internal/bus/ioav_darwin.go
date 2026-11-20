package bus

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdint.h>
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>

typedef CFTypeRef IOAVServiceRef;
extern IOAVServiceRef IOAVServiceCreateWithService(CFAllocatorRef allocator, io_service_t service);
extern IOReturn IOAVServiceReadI2C(IOAVServiceRef service, uint32_t chipAddress, uint32_t offset, void *outputBuffer, uint32_t outputBufferSize);
extern IOReturn IOAVServiceWriteI2C(IOAVServiceRef service, uint32_t chipAddress, uint32_t dataAddress, void *inputBuffer, uint32_t inputBufferSize);

static const char *monctlClass = "DCPAVServiceProxy";

static int monctlIsExternal(io_service_t service) {
	CFTypeRef location = IORegistryEntryCreateCFProperty(service, CFSTR("Location"), kCFAllocatorDefault, 0);
	if (location == NULL) {
		return 0;
	}
	int external = CFGetTypeID(location) == CFStringGetTypeID() &&
		CFStringCompare((CFStringRef)location, CFSTR("External"), 0) == kCFCompareEqualTo;
	CFRelease(location);
	return external;
}

static int monctlList(uint64_t *ids, int max) {
	io_iterator_t it;
	if (IOServiceGetMatchingServices(kIOMainPortDefault, IOServiceMatching(monctlClass), &it) != KERN_SUCCESS) {
		return -1;
	}
	int n = 0;
	io_service_t service;
	while ((service = IOIteratorNext(it)) != IO_OBJECT_NULL) {
		uint64_t id;
		if (n < max && monctlIsExternal(service) &&
			IORegistryEntryGetRegistryEntryID(service, &id) == KERN_SUCCESS) {
			ids[n++] = id;
		}
		IOObjectRelease(service);
	}
	IOObjectRelease(it);
	return n;
}

static IOAVServiceRef monctlOpen(uint64_t id) {
	io_service_t service = IOServiceGetMatchingService(kIOMainPortDefault, IORegistryEntryIDMatching(id));
	if (service == IO_OBJECT_NULL) {
		return NULL;
	}
	IOAVServiceRef av = NULL;
	if (IOObjectConformsTo(service, monctlClass) && monctlIsExternal(service)) {
		av = IOAVServiceCreateWithService(kCFAllocatorDefault, service);
	}
	IOObjectRelease(service);
	return av;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

const maxAVServices = 16

// ErrNoAVService means the display service is gone or is not an external display.
var ErrNoAVService = errors.New("bus: no external display service with this id")

var errAVClosed = errors.New("bus: display service is closed")

// ListAV returns the registry ids of the external display services.
func ListAV() ([]uint64, error) {
	var ids [maxAVServices]C.uint64_t
	n := C.monctlList(&ids[0], maxAVServices)
	if n < 0 {
		return nil, errors.New("bus: cannot list display services")
	}
	out := make([]uint64, n)
	for i := range out {
		out[i] = uint64(ids[i])
	}
	return out, nil
}

type avBus struct {
	service    C.IOAVServiceRef
	edidOffset byte
}

// OpenAV opens the control channel of an external display service.
func OpenAV(id uint64) (Bus, error) {
	service := C.monctlOpen(C.uint64_t(id))
	if service == 0 {
		return nil, ErrNoAVService
	}
	return &avBus{service: service}, nil
}

// Write sends data to addr. The first byte goes out as the data address, as
// the service wants it. On the EDID address it only picks the read offset.
func (b *avBus) Write(addr uint8, data []byte) error {
	if b.service == 0 {
		return errAVClosed
	}
	if len(data) == 0 {
		return errors.New("bus: empty write")
	}
	if addr == AddrEDID {
		b.edidOffset = data[0]
		return nil
	}
	rest := data[1:]
	if r := C.IOAVServiceWriteI2C(b.service, C.uint32_t(addr), C.uint32_t(data[0]), unsafe.Pointer(unsafe.SliceData(rest)), C.uint32_t(len(rest))); r != 0 {
		return fmt.Errorf("bus: write to %#x failed with code %#x", addr, uint32(r))
	}
	return nil
}

func (b *avBus) Read(addr uint8, n int) ([]byte, error) {
	if b.service == 0 {
		return nil, errAVClosed
	}
	if n <= 0 {
		return nil, errors.New("bus: empty read")
	}
	var offset byte
	if addr == AddrEDID {
		offset = b.edidOffset
	}
	out := make([]byte, n)
	if r := C.IOAVServiceReadI2C(b.service, C.uint32_t(addr), C.uint32_t(offset), unsafe.Pointer(&out[0]), C.uint32_t(n)); r != 0 {
		return nil, fmt.Errorf("bus: read from %#x failed with code %#x", addr, uint32(r))
	}
	return out, nil
}

func (b *avBus) Close() error {
	if b.service != 0 {
		C.CFRelease(b.service)
		b.service = 0
	}
	return nil
}

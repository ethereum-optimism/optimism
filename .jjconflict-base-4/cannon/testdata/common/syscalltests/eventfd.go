//go:build linux && mips64
// +build linux,mips64

package syscalltests

import (
	"encoding/binary"
	"fmt"
	"syscall"
)

const (
	EFD_CLOEXEC  = uintptr(0x80000)
	EFD_NONBLOCK = uintptr(0x80)
)

func EventfdTest() {
	// Test a few different valid flag combinations
	fd := callEventfdWithValidFlags(EFD_CLOEXEC | EFD_NONBLOCK)
	fd = callEventfdWithValidFlags(^uintptr(0))
	fd = callEventfdWithValidFlags(EFD_NONBLOCK)

	// Test read/write
	writeToEventObject(fd)
	readFromEventObject(fd)

	// Test invalid flags
	callEventFdWithInvalidFlags(0)
	callEventFdWithInvalidFlags(^EFD_NONBLOCK)
	callEventFdWithInvalidFlags(EFD_CLOEXEC)

	fmt.Println("done")
}

func callEventfdWithValidFlags(flags uintptr) int {
	fmt.Printf("call eventfd with valid flags: '0x%X'\n", flags)

	r1, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, uintptr(0), flags, 0)
	if errno != 0 {
		panic("eventfd2 call failed")
	}
	fd := int(r1)
	fmt.Printf("eventfd2 fd = '%d'\n", fd)

	return fd
}

func callEventFdWithInvalidFlags(flags uintptr) {
	fmt.Printf("call eventfd with invalid flags: '0x%X'\n", flags)

	r1, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, uintptr(0), flags, 0)
	if errno != syscall.EINVAL {
		panic(fmt.Sprintf("expected error EINVAL but got: %v", errno))
	}
	if r1 != ^uintptr(0) {
		panic(fmt.Sprintf("expected r1 to be -1 but got %d", r1))
	}
}

func writeToEventObject(fd int) {
	fmt.Println("write to eventfd object")

	writeVal := uint64(1)
	var writeBuf [8]byte
	binary.BigEndian.PutUint64(writeBuf[:], writeVal)
	n, err := syscall.Write(fd, writeBuf[:])

	validateReadWriteResponse(n, err)
}

func readFromEventObject(fd int) {
	fmt.Println("read from eventfd object")

	var buf [8]byte
	n, err := syscall.Read(fd, buf[:])

	validateReadWriteResponse(n, err)
}

func validateReadWriteResponse(n int, err error) {
	if err != syscall.EAGAIN {
		panic(fmt.Sprintf("expected error EAGAIN but got: %v", err))
	}
	expectedN := -1
	if n != expectedN {
		panic(fmt.Sprintf("expected n=%d but got: %d", expectedN, n))
	}
}

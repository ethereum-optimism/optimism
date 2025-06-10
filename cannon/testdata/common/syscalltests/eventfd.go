//go:build linux && mips64
// +build linux,mips64

package syscalltests

import (
	"encoding/binary"
	"fmt"
	"syscall"
)

func EventfdTest() {
	fd := callEventfd()
	writeToEventObject(fd)
	readFromEventObject(fd)

	fmt.Println("done")
}

const (
	EFD_CLOEXEC  = 0x80000
	EFD_NONBLOCK = 0x80
)

func callEventfd() int {
	fmt.Println("call eventfd")

	flags := EFD_CLOEXEC | EFD_NONBLOCK
	r1, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, uintptr(0), uintptr(flags), 0)
	if errno != 0 {
		panic("eventfd2 call failed")
	}
	fd := int(r1)
	fmt.Printf("eventfd2 fd = %d\n", fd)

	return fd
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

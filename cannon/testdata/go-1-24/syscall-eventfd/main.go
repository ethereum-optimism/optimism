//go:build linux && mips64
// +build linux,mips64

package main

import (
	"encoding/binary"
	"fmt"
	"syscall"
)

func main() {
	eventfd()
}

func eventfd() {
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
	const (
		initVal   = 0
		flags     = EFD_CLOEXEC | EFD_NONBLOCK
		sysEvent2 = syscall.SYS_EVENTFD2
	)

	r1, _, errno := syscall.Syscall(sysEvent2,
		uintptr(initVal),
		uintptr(flags),
		0,
	)
	if errno != 0 {
		panic("eventfd2 call failed")
	}
	fd := int(r1)
	fmt.Printf("eventfd2 fd = %d\n", fd)

	return fd
}

func writeToEventObject(fd int) {
	fmt.Println("write to eventfd object")

	writeVal := uint64(2)
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

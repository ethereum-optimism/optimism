package testutil

import "encoding/binary"

func Uint32ToBytes(val uint32) []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, val)

	return data
}

func Uint64ToBytes(val uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, val)

	return data
}

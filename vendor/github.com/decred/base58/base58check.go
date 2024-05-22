// Copyright (c) 2013-2014 The btcsuite developers
// Copyright (c) 2015-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package base58

import (
	"bytes"
	"errors"

	"github.com/decred/dcrd/crypto/blake256"
)

// ErrChecksum indicates that the checksum of a check-encoded string does not verify against
// the checksum.
var ErrChecksum = errors.New("checksum error")

// ErrInvalidFormat indicates that the check-encoded string has an invalid format.
var ErrInvalidFormat = errors.New("invalid format: version and/or checksum bytes missing")

// checksum returns the first four bytes of BLAKE256(BLAKE256(input)).
func checksum(input []byte) [4]byte {
	var calculatedChecksum [4]byte
	intermediateHash := blake256.Sum256(input)
	finalHash := blake256.Sum256(intermediateHash[:])
	copy(calculatedChecksum[:], finalHash[:])
	return calculatedChecksum
}

// CheckEncode prepends two version bytes and appends a four byte checksum.
func CheckEncode(input []byte, version [2]byte) string {
	b := make([]byte, 0, 2+len(input)+4)
	b = append(b, version[:]...)
	b = append(b, input...)
	calculatedChecksum := checksum(b)
	b = append(b, calculatedChecksum[:]...)
	return Encode(b)
}

// CheckDecode decodes a string that was encoded with CheckEncode and verifies
// the checksum.
func CheckDecode(input string) ([]byte, [2]byte, error) {
	decoded := Decode(input)
	if len(decoded) < 6 {
		return nil, [2]byte{0, 0}, ErrInvalidFormat
	}
	version := [2]byte{decoded[0], decoded[1]}
	dataLen := len(decoded) - 4
	decodedChecksum := decoded[dataLen:]
	calculatedChecksum := checksum(decoded[:dataLen])
	if !bytes.Equal(decodedChecksum, calculatedChecksum[:]) {
		return nil, [2]byte{0, 0}, ErrChecksum
	}
	payload := decoded[2:dataLen]
	return payload, version, nil
}

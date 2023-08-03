package processor

import (
	"encoding/binary"
	"math/big"
)

// DecodeVersionNonce is an re-implementation of Encoding.sol#decodeVersionedNonce.
// If the nonce is greater than 32 bytes (solidity uint256), bytes [32:] are ignored
func DecodeVersionedNonce(nonce *big.Int) (uint16, *big.Int) {
	nonceBytes := nonce.Bytes()
	nonceByteLen := len(nonceBytes)
	if nonceByteLen < 30 {
		// version is 0x0000
		return 0, nonce
	} else if nonceByteLen == 31 {
		// version is 0x00[01..ff]
		return uint16(nonceBytes[0]), new(big.Int).SetBytes(nonceBytes[1:])
	} else {
		// fully specified
		version := binary.BigEndian.Uint16(nonceBytes[:2])
		return version, new(big.Int).SetBytes(nonceBytes[2:])
	}
}

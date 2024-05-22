package superchain

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"golang.org/x/crypto/sha3"
)

// Util-types for hex-encoding/decoding.
// This avoids circular dependencies with downstream eth packages that have their own hex utils.

type Address [20]byte

func (b *Address) UnmarshalText(text []byte) error {
	return decodeHex(b[:], text)
}

func (b Address) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b Address) String() string {
	return encodeHex(b[:])
}

func HexToAddress(s string) Address {
	var a Address
	_ = a.UnmarshalText([]byte(s))
	return a
}

type Hash [32]byte

func has0xPrefix(text []byte) bool {
	return len(text) >= 2 && text[0] == '0' && text[1] == 'x'
}

func decodeHex(dest []byte, text []byte) error {
	if has0xPrefix(text) {
		text = text[2:]
	} else {
		return fmt.Errorf("expected 0x prefix, but got %q", string(text))
	}
	return decodeUnprefixedHex(dest, text)
}

func decodeUnprefixedHex(dest []byte, text []byte) error {
	if len(text) != hex.EncodedLen(len(dest)) {
		return fmt.Errorf("expected %d hex chars, but got %d char input", hex.EncodedLen(len(dest)), len(text))
	}
	_, err := hex.Decode(dest[:], text)
	if err != nil {
		return err
	}
	return nil
}

func encodeHex(bytez []byte) string {
	return "0x" + hex.EncodeToString(bytez[:])
}

func (b *Hash) UnmarshalText(text []byte) error {
	return decodeHex(b[:], text)
}

func (b Hash) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b Hash) String() string {
	return encodeHex(b[:])
}

type HexBytes []byte

func (b *HexBytes) UnmarshalText(text []byte) error {
	if has0xPrefix(text) {
		text = text[2:]
	} else {
		return fmt.Errorf("expected 0x prefix, but got %q", string(text))
	}
	*b = make([]byte, hex.DecodedLen(len(text)))
	return decodeUnprefixedHex((*b)[:], text)
}

func (b HexBytes) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b HexBytes) String() string {
	return encodeHex(b[:])
}

type HexBig big.Int

func (b HexBig) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b HexBig) String() string {
	if sign := (*big.Int)(&b).Sign(); sign == 0 {
		return "0x0"
	} else if sign > 0 {
		return "0x" + (*big.Int)(&b).Text(16)
	} else {
		return "-0x" + (*big.Int)(&b).Text(16)[1:]
	}
}

func (b *HexBig) UnmarshalText(text []byte) error {
	return (*big.Int)(b).UnmarshalText(text)
}

func keccak256(v []byte) Hash {
	st := sha3.NewLegacyKeccak256()
	st.Write(v)
	return *(*[32]byte)(st.Sum(nil))
}

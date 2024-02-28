package subkey

import (
	"encoding/hex"
	"strings"
)

// DecodeHex decodes the hex string to bytes.
// `0x` prefix is accepted.
func DecodeHex(uri string) ([]byte, bool) {
	if strings.HasPrefix(uri, "0x") {
		uri = strings.TrimPrefix(uri, "0x")
	}
	res, err := hex.DecodeString(uri)
	return res, err == nil
}

// EncodeHex encodes bytes to hex
// `0x` prefix is added.
func EncodeHex(b []byte) string {
	res := hex.EncodeToString(b)
	if !strings.HasPrefix(res, "0x") {
		res = "0x" + res
	}

	return res
}

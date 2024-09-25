package crypto

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// EncodePrivKey encodes the given private key in 32 bytes
func EncodePrivKey(priv *ecdsa.PrivateKey) hexutil.Bytes {
	privkey := make([]byte, 32)
	blob := priv.D.Bytes()
	copy(privkey[32-len(blob):], blob)
	return privkey
}

func EncodePrivKeyToString(priv *ecdsa.PrivateKey) string {
	return hexutil.Encode(EncodePrivKey(priv))
}

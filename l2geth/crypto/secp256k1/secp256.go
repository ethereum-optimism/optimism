package secp256k1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

var (
	ErrInvalidMsgLen       = secp256k1.ErrInvalidMsgLen
	ErrInvalidSignatureLen = secp256k1.ErrInvalidSignatureLen
	ErrInvalidRecoveryID   = secp256k1.ErrInvalidRecoveryID
	ErrInvalidKey          = secp256k1.ErrInvalidKey
	ErrInvalidPubkey       = secp256k1.ErrInvalidPubkey
	ErrSignFailed          = secp256k1.ErrSignFailed
	ErrRecoverFailed       = secp256k1.ErrRecoverFailed
)

// Sign creates a recoverable ECDSA signature.
// The produced signature is in the 65-byte [R || S || V] format where V is 0 or 1.
//
// The caller is responsible for ensuring that msg cannot be chosen
// directly by an attacker. It is usually preferable to use a cryptographic
// hash function on any input before handing it to this function.
func Sign(msg []byte, seckey []byte) ([]byte, error) {
	return secp256k1.Sign(msg, seckey)
}

// RecoverPubkey returns the public key of the signer.
// msg must be the 32-byte hash of the message to be signed.
// sig must be a 65-byte compact ECDSA signature containing the
// recovery id as the last element.
func RecoverPubkey(msg []byte, sig []byte) ([]byte, error) {
	return secp256k1.RecoverPubkey(msg, sig)
}

// VerifySignature checks that the given pubkey created signature over message.
// The signature should be in [R || S] format.
func VerifySignature(pubkey, msg, signature []byte) bool {
	return secp256k1.VerifySignature(pubkey, msg, signature)
}

// DecompressPubkey parses a public key in the 33-byte compressed format.
// It returns non-nil coordinates if the public key is valid.
func DecompressPubkey(pubkey []byte) (x, y *big.Int) {
	return secp256k1.DecompressPubkey(pubkey)
}

// CompressPubkey encodes a public key to 33-byte compressed format.
func CompressPubkey(x, y *big.Int) []byte {
	return secp256k1.CompressPubkey(x, y)
}

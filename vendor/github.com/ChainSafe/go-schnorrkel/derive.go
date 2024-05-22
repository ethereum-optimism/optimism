package schnorrkel

import (
	"crypto/rand"
	"errors"

	"github.com/gtank/merlin"
	r255 "github.com/gtank/ristretto255"
)

const ChainCodeLength = 32

var (
	ErrDeriveHardKeyType = errors.New("cannot derive hard key type, DerivableKey must be of type SecretKey")
)

// DerivableKey implements DeriveKey
type DerivableKey interface {
	Encode() [32]byte
	Decode([32]byte) error
	DeriveKey(*merlin.Transcript, [ChainCodeLength]byte) (*ExtendedKey, error)
}

// ExtendedKey consists of a DerivableKey which can be a schnorrkel public or private key
// as well as chain code
type ExtendedKey struct {
	key       DerivableKey
	chaincode [ChainCodeLength]byte
}

// NewExtendedKey creates an ExtendedKey given a DerivableKey and chain code
func NewExtendedKey(k DerivableKey, cc [ChainCodeLength]byte) *ExtendedKey {
	return &ExtendedKey{
		key:       k,
		chaincode: cc,
	}
}

// Key returns the schnorrkel key underlying the ExtendedKey
func (ek *ExtendedKey) Key() DerivableKey {
	return ek.key
}

// ChainCode returns the chain code underlying the ExtendedKey
func (ek *ExtendedKey) ChainCode() [ChainCodeLength]byte {
	return ek.chaincode
}

// Secret returns the SecretKey underlying the ExtendedKey
// if it's not a secret key, it returns an error
func (ek *ExtendedKey) Secret() (*SecretKey, error) {
	if priv, ok := ek.key.(*SecretKey); ok {
		return priv, nil
	}

	return nil, errors.New("extended key is not a secret key")
}

// Public returns the PublicKey underlying the ExtendedKey
func (ek *ExtendedKey) Public() (*PublicKey, error) {
	if pub, ok := ek.key.(*PublicKey); ok {
		return pub, nil
	}

	if priv, ok := ek.key.(*SecretKey); ok {
		return priv.Public()
	}

	return nil, errors.New("extended key is not a valid public or private key")
}

// DeriveKey derives an extended key from an extended key
func (ek *ExtendedKey) DeriveKey(t *merlin.Transcript) (*ExtendedKey, error) {
	return ek.key.DeriveKey(t, ek.chaincode)
}

// HardDeriveMiniSecretKey implements BIP-32 like "hard" derivation of a mini
// secret from an extended key's secret key
func (ek *ExtendedKey) HardDeriveMiniSecretKey(i []byte) (*ExtendedKey, error) {
	sk, err := ek.Secret()
	if err != nil {
		return nil, err
	}

	msk, chainCode, err := sk.HardDeriveMiniSecretKey(i, ek.chaincode)
	if err != nil {
		return nil, err
	}

	return NewExtendedKey(msk, chainCode), nil
}

// DeriveKeyHard derives a Hard subkey identified by the byte array i and chain
// code
func DeriveKeyHard(key DerivableKey, i []byte, cc [ChainCodeLength]byte) (*ExtendedKey, error) {
	switch k := key.(type) {
	case *SecretKey:
		msk, resCC, err := k.HardDeriveMiniSecretKey(i, cc)
		if err != nil {
			return nil, err
		}
		return NewExtendedKey(msk.ExpandEd25519(), resCC), nil

	default:
		return nil, ErrDeriveHardKeyType
	}
}

// DerviveKeySoft is an alias for DervieKeySimple() used to derive a Soft subkey
// identified by the byte array i and chain code
func DeriveKeySoft(key DerivableKey, i []byte, cc [ChainCodeLength]byte) (*ExtendedKey, error) {
	return DeriveKeySimple(key, i, cc)
}

// DeriveKeySimple derives a Soft subkey identified by byte array i and chain code.
func DeriveKeySimple(key DerivableKey, i []byte, cc [ChainCodeLength]byte) (*ExtendedKey, error) {
	t := merlin.NewTranscript("SchnorrRistrettoHDKD")
	t.AppendMessage([]byte("sign-bytes"), i)
	return key.DeriveKey(t, cc)
}

// DeriveKey derives a new secret key and chain code from an existing secret key and chain code
func (sk *SecretKey) DeriveKey(t *merlin.Transcript, cc [ChainCodeLength]byte) (*ExtendedKey, error) {
	pub, err := sk.Public()
	if err != nil {
		return nil, err
	}

	sc, dcc, err := pub.DeriveScalarAndChaincode(t, cc)
	if err != nil {
		return nil, err
	}

	// TODO: need transcript RNG to match rust-schnorrkel
	// see: https://github.com/w3f/schnorrkel/blob/798ab3e0813aa478b520c5cf6dc6e02fd4e07f0a/src/derive.rs#L186
	nonce := [32]byte{}
	_, err = rand.Read(nonce[:])
	if err != nil {
		return nil, err
	}

	dsk, err := ScalarFromBytes(sk.key)
	if err != nil {
		return nil, err
	}

	dsk.Add(dsk, sc)

	dskBytes := [32]byte{}
	copy(dskBytes[:], dsk.Encode([]byte{}))

	skNew := &SecretKey{
		key:   dskBytes,
		nonce: nonce,
	}

	return &ExtendedKey{
		key:       skNew,
		chaincode: dcc,
	}, nil
}

// HardDeriveMiniSecretKey implements BIP-32 like "hard" derivation of a mini
// secret from a secret key
func (sk *SecretKey) HardDeriveMiniSecretKey(i []byte, cc [ChainCodeLength]byte) (
	*MiniSecretKey, [ChainCodeLength]byte, error) {

	t := merlin.NewTranscript("SchnorrRistrettoHDKD")
	t.AppendMessage([]byte("sign-bytes"), i)
	t.AppendMessage([]byte("chain-code"), cc[:])
	skenc := sk.Encode()
	t.AppendMessage([]byte("secret-key"), skenc[:])

	msk := [MiniSecretKeySize]byte{}
	mskBytes := t.ExtractBytes([]byte("HDKD-hard"), MiniSecretKeySize)
	copy(msk[:], mskBytes)

	ccRes := [ChainCodeLength]byte{}
	ccBytes := t.ExtractBytes([]byte("HDKD-chaincode"), ChainCodeLength)
	copy(ccRes[:], ccBytes)

	miniSec, err := NewMiniSecretKeyFromRaw(msk)

	return miniSec, ccRes, err
}

// HardDeriveMiniSecretKey implements BIP-32 like "hard" derivation of a mini
// secret from a mini secret key
func (mk *MiniSecretKey) HardDeriveMiniSecretKey(i []byte, cc [ChainCodeLength]byte) (
	*MiniSecretKey, [ChainCodeLength]byte, error) {
	sk := mk.ExpandEd25519()
	return sk.HardDeriveMiniSecretKey(i, cc)
}

// DeriveKey derives an Extended Key from the Mini Secret Key
func (mk *MiniSecretKey) DeriveKey(t *merlin.Transcript, cc [ChainCodeLength]byte) (*ExtendedKey, error) {
	if t == nil {
		return nil, errors.New("transcript provided is nil")
	}

	sk := mk.ExpandEd25519()
	return sk.DeriveKey(t, cc)
}

func (pk *PublicKey) DeriveKey(t *merlin.Transcript, cc [ChainCodeLength]byte) (*ExtendedKey, error) {
	if t == nil {
		return nil, errors.New("transcript provided is nil")
	}

	sc, dcc, err := pk.DeriveScalarAndChaincode(t, cc)
	if err != nil {
		return nil, err
	}

	// derivedPk = pk + (sc * g)
	p1 := r255.NewElement().ScalarBaseMult(sc)
	p2 := r255.NewElement()
	p2.Add(pk.key, p1)

	pub := &PublicKey{
		key: p2,
	}

	return &ExtendedKey{
		key:       pub,
		chaincode: dcc,
	}, nil
}

// DeriveScalarAndChaincode derives a new scalar and chain code from an existing public key and chain code
func (pk *PublicKey) DeriveScalarAndChaincode(t *merlin.Transcript, cc [ChainCodeLength]byte) (*r255.Scalar, [ChainCodeLength]byte, error) {
	if t == nil {
		return nil, [ChainCodeLength]byte{}, errors.New("transcript provided is nil")
	}

	t.AppendMessage([]byte("chain-code"), cc[:])
	pkenc := pk.Encode()
	t.AppendMessage([]byte("public-key"), pkenc[:])

	scBytes := t.ExtractBytes([]byte("HDKD-scalar"), 64)
	sc := r255.NewScalar()
	sc.FromUniformBytes(scBytes)

	ccBytes := t.ExtractBytes([]byte("HDKD-chaincode"), ChainCodeLength)
	ccRes := [ChainCodeLength]byte{}
	copy(ccRes[:], ccBytes)
	return sc, ccRes, nil
}

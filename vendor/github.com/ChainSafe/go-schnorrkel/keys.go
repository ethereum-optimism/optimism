package schnorrkel

import (
	"crypto/rand"
	"crypto/sha512"
	"errors"

	"github.com/gtank/merlin"
	r255 "github.com/gtank/ristretto255"
)

const (
	// MiniSecretKeySize is the length in bytes of a MiniSecretKey
	MiniSecretKeySize = 32

	// SecretKeySize is the length in bytes of a SecretKey
	SecretKeySize = 32

	// PublicKeySize is the length in bytes of a PublicKey
	PublicKeySize = 32
)

var (
	publicKeyAtInfinity    = r255.NewElement().ScalarBaseMult(r255.NewScalar())
	errPublicKeyAtInfinity = errors.New("public key is the point at infinity")
)

// MiniSecretKey is a secret scalar
type MiniSecretKey struct {
	key [MiniSecretKeySize]byte
}

// SecretKey consists of a secret scalar and a signing nonce
type SecretKey struct {
	key   [32]byte // TODO: change this to a *r255.Scalar
	nonce [32]byte
}

// PublicKey is a field element
type PublicKey struct {
	key *r255.Element
}

// GenerateKeypair generates a new schnorrkel secret key and public key
func GenerateKeypair() (*SecretKey, *PublicKey, error) {
	// decodes priv bytes as little-endian
	msc, err := GenerateMiniSecretKey()
	if err != nil {
		return nil, nil, err
	}
	return msc.ExpandEd25519(), msc.Public(), nil
}

// NewMiniSecretKey derives a mini secret key from a seed
func NewMiniSecretKey(b [64]byte) *MiniSecretKey {
	s := r255.NewScalar()
	s.FromUniformBytes(b[:])
	enc := s.Encode([]byte{})
	sk := [MiniSecretKeySize]byte{}
	copy(sk[:], enc)
	return &MiniSecretKey{key: sk}
}

// NewMiniSecretKeyFromRaw derives a mini secret key from little-endian encoded raw bytes.
func NewMiniSecretKeyFromRaw(b [MiniSecretKeySize]byte) (*MiniSecretKey, error) {
	s := b
	return &MiniSecretKey{key: s}, nil
}

// NewMiniSecretKeyFromHex returns a new MiniSecretKey from the given hex-encoded string
func NewMiniSecretKeyFromHex(s string) (*MiniSecretKey, error) {
	b, err := HexToBytes(s)
	if err != nil {
		return nil, err
	}

	pk := [32]byte{}
	copy(pk[:], b)

	priv, err := NewMiniSecretKeyFromRaw(pk)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

// GenerateMiniSecretKey generates a mini secret key from random
func GenerateMiniSecretKey() (*MiniSecretKey, error) {
	s := [MiniSecretKeySize]byte{}
	_, err := rand.Read(s[:])
	if err != nil {
		return nil, err
	}

	return &MiniSecretKey{key: s}, nil
}

// NewSecretKey creates a new secret key from input bytes
func NewSecretKey(key [SecretKeySize]byte, nonce [32]byte) *SecretKey {
	return &SecretKey{
		key:   key,
		nonce: nonce,
	}
}

// NewPublicKey creates a new public key from input bytes
func NewPublicKey(b [PublicKeySize]byte) (*PublicKey, error) {
	e := r255.NewElement()
	err := e.Decode(b[:])
	if err != nil {
		return nil, err
	}

	return &PublicKey{
		key: e,
	}, nil
}

// NewPublicKeyFromHex returns a PublicKey from a hex-encoded string
func NewPublicKeyFromHex(s string) (*PublicKey, error) {
	pubhex, err := HexToBytes(s)
	if err != nil {
		return nil, err
	}

	in := [32]byte{}
	copy(in[:], pubhex)

	pub := &PublicKey{}
	err = pub.Decode(in)
	if err != nil {
		return nil, err
	}

	return pub, nil
}

// Decode creates a MiniSecretKey from the given input
func (s *MiniSecretKey) Decode(in [MiniSecretKeySize]byte) error {
	msc, err := NewMiniSecretKeyFromRaw(in)
	if err != nil {
		return err
	}

	s.key = msc.key
	return nil
}

// Encode returns the MiniSecretKey's underlying bytes
func (s *MiniSecretKey) Encode() [MiniSecretKeySize]byte {
	return s.key
}

// ExpandUniform expands a MiniSecretKey into a SecretKey
func (s *MiniSecretKey) ExpandUniform() *SecretKey {
	t := merlin.NewTranscript("ExpandSecretKeys")
	t.AppendMessage([]byte("mini"), s.key[:])
	scalarBytes := t.ExtractBytes([]byte("sk"), 64)
	key := r255.NewScalar()
	key.FromUniformBytes(scalarBytes[:])
	nonce := t.ExtractBytes([]byte("no"), 32)
	key32 := [32]byte{}
	copy(key32[:], key.Encode([]byte{}))
	nonce32 := [32]byte{}
	copy(nonce32[:], nonce)
	return &SecretKey{
		key:   key32,
		nonce: nonce32,
	}
}

// ExpandEd25519 expands a MiniSecretKey into a SecretKey using ed25519-style bit clamping
// https://github.com/w3f/schnorrkel/blob/43f7fc00724edd1ef53d5ae13d82d240ed6202d5/src/keys.rs#L196
func (s *MiniSecretKey) ExpandEd25519() *SecretKey {
	h := sha512.Sum512(s.key[:])
	sk := &SecretKey{
		key:   [SecretKeySize]byte{},
		nonce: [32]byte{},
	}

	copy(sk.key[:], h[:32])

	sk.key[0] &= 248
	sk.key[31] &= 63
	sk.key[31] |= 64
	t := divideScalarByCofactor(sk.key[:])

	copy(sk.key[:], t)
	copy(sk.nonce[:], h[32:])
	return sk
}

// Public returns the PublicKey corresponding to this MiniSecretKey
func (s *MiniSecretKey) Public() *PublicKey {
	e := r255.NewElement()
	sk := s.ExpandEd25519()
	skey, err := ScalarFromBytes(sk.key)
	if err != nil {
		return nil
	}

	return &PublicKey{key: e.ScalarBaseMult(skey)}
}

// Decode creates a SecretKey from the given input
func (s *SecretKey) Decode(in [SecretKeySize]byte) error {
	s.key = in
	return nil
}

// Encode returns the SecretKey's underlying bytes
func (s *SecretKey) Encode() [SecretKeySize]byte {
	return s.key
}

// Public gets the public key corresponding to this SecretKey
func (s *SecretKey) Public() (*PublicKey, error) {
	e := r255.NewElement()
	sc, err := ScalarFromBytes(s.key)
	if err != nil {
		return nil, err
	}
	return &PublicKey{key: e.ScalarBaseMult(sc)}, nil
}

// Decode creates a PublicKey from the given input
func (p *PublicKey) Decode(in [PublicKeySize]byte) error {
	p.key = r255.NewElement()
	return p.key.Decode(in[:])
}

// Encode returns the encoded point underlying the public key
func (p *PublicKey) Encode() [PublicKeySize]byte {
	b := p.key.Encode([]byte{})
	enc := [PublicKeySize]byte{}
	copy(enc[:], b)
	return enc
}

package schnorrkel

import (
	"errors"

	"github.com/gtank/merlin"
	r255 "github.com/gtank/ristretto255"
)

// SignatureSize is the length in bytes of a signature
const SignatureSize = 64

// ErrSignatureNotMarkedSchnorrkel is returned when attempting to decode a signature that is not marked as schnorrkel
var ErrSignatureNotMarkedSchnorrkel = errors.New("signature is not marked as a schnorrkel signature")

// Signature holds a schnorrkel signature
type Signature struct {
	r *r255.Element
	s *r255.Scalar
}

// NewSignatureFromHex returns a new Signature from the given hex-encoded string
func NewSignatureFromHex(s string) (*Signature, error) {
	sighex, err := HexToBytes(s)
	if err != nil {
		return nil, err
	}

	sigin := [64]byte{}
	copy(sigin[:], sighex)

	sig := &Signature{}
	err = sig.Decode(sigin)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

// NewSigningContext returns a new transcript initialized with the context for the signature
//.see: https://github.com/w3f/schnorrkel/blob/db61369a6e77f8074eb3247f9040ccde55697f20/src/context.rs#L183
func NewSigningContext(context, msg []byte) *merlin.Transcript {
	t := merlin.NewTranscript("SigningContext")
	t.AppendMessage([]byte(""), context)
	t.AppendMessage([]byte("sign-bytes"), msg)
	return t
}

// Sign uses the schnorr signature algorithm to sign a message
// See the following for the transcript message
// https://github.com/w3f/schnorrkel/blob/db61369a6e77f8074eb3247f9040ccde55697f20/src/sign.rs#L158
// Schnorr w/ transcript, secret key x:
// 1. choose random r from group
// 2. R = gr
// 3. k = scalar(transcript.extract_bytes())
// 4. s = kx + r
// signature: (R, s)
// public key used for verification: y = g^x
func (sk *SecretKey) Sign(t *merlin.Transcript) (*Signature, error) {
	t.AppendMessage([]byte("proto-name"), []byte("Schnorr-sig"))

	pub, err := sk.Public()
	if err != nil {
		return nil, err
	}
	pubc := pub.Encode()

	t.AppendMessage([]byte("sign:pk"), pubc[:])

	// note: TODO: merlin library doesn't have build_rng yet.
	// see https://github.com/w3f/schnorrkel/blob/798ab3e0813aa478b520c5cf6dc6e02fd4e07f0a/src/context.rs#L153
	// r := t.ExtractBytes([]byte("signing"), 32)

	// choose random r (nonce)
	r, err := NewRandomScalar()
	if err != nil {
		return nil, err
	}
	R := r255.NewElement().ScalarBaseMult(r)
	t.AppendMessage([]byte("sign:R"), R.Encode([]byte{}))

	// form k
	kb := t.ExtractBytes([]byte("sign:c"), 64)
	k := r255.NewScalar()
	k.FromUniformBytes(kb)

	// form scalar from secret key x
	x, err := ScalarFromBytes(sk.key)
	if err != nil {
		return nil, err
	}

	// s = kx + r
	s := x.Multiply(x, k).Add(x, r)

	return &Signature{
		r: R,
		s: s,
	}, nil
}

// Verify verifies a schnorr signature with format: (R, s) where y is the public key
// 1. k = scalar(transcript.extract_bytes())
// 2. R' = -ky + gs
// 3. return R' == R
func (p *PublicKey) Verify(s *Signature, t *merlin.Transcript) (bool, error) {
	if s == nil {
		return false, errors.New("signature provided is nil")
	}

	if t == nil {
		return false, errors.New("transcript provided is nil")
	}

	if p.key.Equal(publicKeyAtInfinity) == 1 {
		return false, errPublicKeyAtInfinity
	}

	t.AppendMessage([]byte("proto-name"), []byte("Schnorr-sig"))
	pubc := p.Encode()
	t.AppendMessage([]byte("sign:pk"), pubc[:])
	t.AppendMessage([]byte("sign:R"), s.r.Encode([]byte{}))

	kb := t.ExtractBytes([]byte("sign:c"), 64)
	k := r255.NewScalar()
	k.FromUniformBytes(kb)

	Rp := r255.NewElement()
	Rp = Rp.ScalarBaseMult(s.s)
	ky := r255.NewElement().ScalarMult(k, p.key)
	Rp = Rp.Subtract(Rp, ky)

	return Rp.Equal(s.r) == 1, nil
}

// Decode sets a Signature from bytes
// see: https://github.com/w3f/schnorrkel/blob/db61369a6e77f8074eb3247f9040ccde55697f20/src/sign.rs#L100
func (s *Signature) Decode(in [SignatureSize]byte) error {
	if in[63]&128 == 0 {
		return ErrSignatureNotMarkedSchnorrkel
	}

	cp := [64]byte{}
	copy(cp[:], in[:])

	s.r = r255.NewElement()
	err := s.r.Decode(cp[:32])
	if err != nil {
		return err
	}

	cp[63] &= 127
	s.s = r255.NewScalar()
	return s.s.Decode(cp[32:])
}

// Encode turns a signature into a byte array
// see: https://github.com/w3f/schnorrkel/blob/db61369a6e77f8074eb3247f9040ccde55697f20/src/sign.rs#L77
func (s *Signature) Encode() [SignatureSize]byte {
	out := [64]byte{}
	renc := s.r.Encode([]byte{})
	copy(out[:32], renc)
	senc := s.s.Encode([]byte{})
	copy(out[32:], senc)
	out[63] |= 128
	return out
}

// DecodeNotDistinguishedFromEd25519 sets a signature from bytes, not checking if the signature
// is explicitly marked as a schnorrkel signature
func (s *Signature) DecodeNotDistinguishedFromEd25519(in [SignatureSize]byte) error {
	cp := [64]byte{}
	copy(cp[:], in[:])
	cp[63] |= 128
	return s.Decode(cp)
}

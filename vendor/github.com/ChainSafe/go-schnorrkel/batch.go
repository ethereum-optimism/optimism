package schnorrkel

import (
	"errors"

	"github.com/gtank/merlin"
	r255 "github.com/gtank/ristretto255"
)

// VerifyBatch batch verifies the given signatures
func VerifyBatch(transcripts []*merlin.Transcript, signatures []*Signature, pubkeys []*PublicKey) (bool, error) {
	if len(transcripts) != len(signatures) || len(signatures) != len(pubkeys) || len(pubkeys) != len(transcripts) {
		return false, errors.New("the number of transcripts, signatures, and public keys must be equal")
	}

	if len(transcripts) == 0 {
		return true, nil
	}

	var err error
	zero := r255.NewElement().Zero()
	zs := make([]*r255.Scalar, len(transcripts))
	for i := range zs {
		zs[i], err = NewRandomScalar()
		if err != nil {
			return false, err
		}
	}

	// compute H(R_i || P_i || m_i)
	hs := make([]*r255.Scalar, len(transcripts))
	s := make([]r255.Scalar, len(transcripts))
	for i, t := range transcripts {
		if t == nil {
			return false, errors.New("transcript provided was nil")
		}

		t.AppendMessage([]byte("proto-name"), []byte("Schnorr-sig"))
		pubc := pubkeys[i].Encode()
		t.AppendMessage([]byte("sign:pk"), pubc[:])
		t.AppendMessage([]byte("sign:R"), signatures[i].r.Encode([]byte{}))

		h := t.ExtractBytes([]byte("sign:c"), 64)
		s[i] = *r255.NewScalar()
		hs[i] = &s[i]
		hs[i].FromUniformBytes(h)
	}

	// compute ∑ z_i P_i H(R_i || P_i || m_i)
	ps := make([]*r255.Element, len(pubkeys))
	for i, p := range pubkeys {
		if p == nil {
			return false, errors.New("public key provided was nil")
		}

		ps[i] = r255.NewElement().ScalarMult(zs[i], p.key)
	}

	phs := r255.NewElement().MultiScalarMult(hs, ps)

	// compute ∑ z_i s_i and ∑ z_i R_i
	ss := r255.NewScalar()
	rs := r255.NewElement()
	for i, s := range signatures {
		if s == nil {
			return false, errors.New("signature provided was nil")
		}

		zsi := r255.NewScalar().Multiply(s.s, zs[i])
		ss = r255.NewScalar().Add(ss, zsi)
		zri := r255.NewElement().ScalarMult(zs[i], s.r)
		rs = r255.NewElement().Add(rs, zri)
	}

	// ∑ z_i P_i H(R_i || P_i || m_i) + ∑ R_i
	z := r255.NewElement().Add(phs, rs)

	// B ∑ z_i  s_i
	sb := r255.NewElement().ScalarBaseMult(ss)

	// check  -B ∑ z_i s_i + ∑ z_i P_i H(R_i || P_i || m_i) + ∑ z_i R_i = 0
	sb_neg := r255.NewElement().Negate(sb)
	res := r255.NewElement().Add(sb_neg, z)

	return res.Equal(zero) == 1, nil
}

type BatchVerifier struct {
	hs      []*r255.Scalar  // transcript scalar
	ss      *r255.Scalar    // sum of signature.S: ∑ z_i s_i
	rs      *r255.Element   // sum of signature.R: ∑ z_i R_i
	pubkeys []*r255.Element // z_i P_i
}

func NewBatchVerifier() *BatchVerifier {
	return &BatchVerifier{
		hs:      []*r255.Scalar{},
		ss:      r255.NewScalar(),
		rs:      r255.NewElement(),
		pubkeys: []*r255.Element{},
	}
}

func (v *BatchVerifier) Add(t *merlin.Transcript, sig *Signature, pubkey *PublicKey) error {
	if t == nil {
		return errors.New("provided transcript is nil")
	}

	if sig == nil {
		return errors.New("provided signature is nil")
	}

	if pubkey == nil {
		return errors.New("provided public key is nil")
	}

	z, err := NewRandomScalar()
	if err != nil {
		return err
	}

	t.AppendMessage([]byte("proto-name"), []byte("Schnorr-sig"))
	pubc := pubkey.Encode()
	t.AppendMessage([]byte("sign:pk"), pubc[:])
	t.AppendMessage([]byte("sign:R"), sig.r.Encode([]byte{}))

	h := t.ExtractBytes([]byte("sign:c"), 64)
	s := r255.NewScalar()
	s.FromUniformBytes(h)
	v.hs = append(v.hs, s)

	zs := r255.NewScalar().Multiply(z, sig.s)
	v.ss.Add(v.ss, zs)
	zr := r255.NewElement().ScalarMult(z, sig.r)
	v.rs.Add(v.rs, zr)

	p := r255.NewElement().ScalarMult(z, pubkey.key)
	v.pubkeys = append(v.pubkeys, p)
	return nil
}

func (v *BatchVerifier) Verify() bool {
	zero := r255.NewElement().Zero()

	// compute ∑ z_i P_i H(R_i || P_i || m_i)
	phs := r255.NewElement().MultiScalarMult(v.hs, v.pubkeys)

	// ∑ z_i P_i H(R_i || P_i || m_i) + ∑ z_i R_i
	z := r255.NewElement().Add(phs, v.rs)

	// B ∑ z_i s_i
	sb := r255.NewElement().ScalarBaseMult(v.ss)

	// check  -B ∑ z_i s_i + ∑ z_i P_i H(R_i || P_i || m_i) + ∑ z_i R_i = 0
	sb_neg := r255.NewElement().Negate(sb)
	res := r255.NewElement().Add(sb_neg, z)

	return res.Equal(zero) == 1
}

package sr25519

import (
	"errors"

	sr25519 "github.com/ChainSafe/go-schnorrkel"
	"github.com/gtank/merlin"
	"github.com/vedhavyas/go-subkey"
)

const (
	miniSecretKeyLength = 32

	secretKeyLength = 64

	signatureLength = 64
)

type keyRing struct {
	seed   []byte
	secret *sr25519.SecretKey
	pub    *sr25519.PublicKey
}

func (kr keyRing) Sign(msg []byte) (signature []byte, err error) {
	sig, err := kr.secret.Sign(signingContext(msg))
	if err != nil {
		return signature, err
	}

	s := sig.Encode()
	return s[:], nil
}

func (kr keyRing) Verify(msg []byte, signature []byte) bool {
	var sigs [signatureLength]byte
	copy(sigs[:], signature)
	sig := new(sr25519.Signature)
	if err := sig.Decode(sigs); err != nil {
		return false
	}
	ok, err := kr.pub.Verify(sig, signingContext(msg))
	if err != nil || !ok {
		return false
	}

	return true
}

func signingContext(msg []byte) *merlin.Transcript {
	return sr25519.NewSigningContext([]byte("substrate"), msg)
}

// Public returns the public key in bytes
func (kr keyRing) Public() []byte {
	pub := kr.pub.Encode()
	return pub[:]
}

func (kr keyRing) Seed() []byte {
	return kr.seed
}

func (kr keyRing) AccountID() []byte {
	return kr.Public()
}

func (kr keyRing) SS58Address(network uint8) (string, error) {
	return subkey.SS58Address(kr.AccountID(), network)
}

func (kr keyRing) SS58AddressWithAccountIDChecksum(network uint8) (string, error) {
	return subkey.SS58AddressWithAccountIDChecksum(kr.AccountID(), network)
}

func deriveKeySoft(secret *sr25519.SecretKey, cc [32]byte) (*sr25519.SecretKey, error) {
	t := merlin.NewTranscript("SchnorrRistrettoHDKD")
	t.AppendMessage([]byte("sign-bytes"), nil)
	ek, err := secret.DeriveKey(t, cc)
	if err != nil {
		return nil, err
	}
	return ek.Secret()
}

func deriveKeyHard(secret *sr25519.SecretKey, cc [32]byte) (*sr25519.MiniSecretKey, error) {
	t := merlin.NewTranscript("SchnorrRistrettoHDKD")
	t.AppendMessage([]byte("sign-bytes"), nil)
	t.AppendMessage([]byte("chain-code"), cc[:])
	s := secret.Encode()
	t.AppendMessage([]byte("secret-key"), s[:])
	mskb := t.ExtractBytes([]byte("HDKD-hard"), miniSecretKeyLength)
	msk := [miniSecretKeyLength]byte{}
	copy(msk[:], mskb)
	return sr25519.NewMiniSecretKeyFromRaw(msk)
}

type Scheme struct{}

func (s Scheme) String() string {
	return "Sr25519"
}

func (s Scheme) Generate() (subkey.KeyPair, error) {
	ms, err := sr25519.GenerateMiniSecretKey()
	if err != nil {
		return nil, err
	}

	secret := ms.ExpandEd25519()
	pub, err := secret.Public()
	if err != nil {
		return nil, err
	}

	seed := ms.Encode()
	return keyRing{
		seed:   seed[:],
		secret: secret,
		pub:    pub,
	}, nil
}

func (s Scheme) FromSeed(seed []byte) (subkey.KeyPair, error) {
	switch len(seed) {
	case miniSecretKeyLength:
		var mss [32]byte
		copy(mss[:], seed)
		ms, err := sr25519.NewMiniSecretKeyFromRaw(mss)
		if err != nil {
			return nil, err
		}

		return keyRing{
			seed:   seed,
			secret: ms.ExpandEd25519(),
			pub:    ms.Public(),
		}, nil

	case secretKeyLength:
		var key, nonce [32]byte
		copy(key[:], seed[0:32])
		copy(nonce[:], seed[32:64])
		secret := sr25519.NewSecretKey(key, nonce)
		pub, err := secret.Public()
		if err != nil {
			return nil, err
		}

		return keyRing{
			seed:   seed,
			secret: secret,
			pub:    pub,
		}, nil
	}

	return nil, errors.New("invalid seed length")
}

func (s Scheme) FromPhrase(phrase, pwd string) (subkey.KeyPair, error) {
	ms, err := sr25519.MiniSecretKeyFromMnemonic(phrase, pwd)
	if err != nil {
		return nil, err
	}

	secret := ms.ExpandEd25519()
	pub, err := secret.Public()
	if err != nil {
		return nil, err
	}

	seed := ms.Encode()
	return keyRing{
		seed:   seed[:],
		secret: secret,
		pub:    pub,
	}, nil
}

func (s Scheme) Derive(pair subkey.KeyPair, djs []subkey.DeriveJunction) (subkey.KeyPair, error) {
	kr := pair.(keyRing)
	secret := kr.secret
	seed := kr.seed
	var err error
	for _, dj := range djs {
		if dj.IsHard {
			ms, err := deriveKeyHard(secret, dj.ChainCode)
			if err != nil {
				return nil, err
			}

			secret = ms.ExpandEd25519()
			if seed != nil {
				es := ms.Encode()
				seed = es[:]
			}
			continue
		}

		secret, err = deriveKeySoft(secret, dj.ChainCode)
		if err != nil {
			return nil, err
		}
		seed = nil
	}

	pub, err := secret.Public()
	if err != nil {
		return nil, err
	}

	return &keyRing{seed: seed, secret: secret, pub: pub}, nil
}

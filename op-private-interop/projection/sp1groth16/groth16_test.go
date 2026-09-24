package sp1groth16

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	VKey         common.Hash   `json:"vkey"`
	PublicValues hexutil.Bytes `json:"publicValues"`
	Proof        hexutil.Bytes `json:"proof"`
}

// fixtureVKRoot is the recursion vk root embedded in the upstream v6.0.0 fixture proof.
var fixtureVKRoot = common.HexToHash("0x008cd56e10c2fe24795cff1e1d1f40d3a324528d315674da45d26afb376e8670")

func loadFixture(t *testing.T) (Circuit, fixture) {
	t.Helper()
	vk, err := os.ReadFile("testdata/groth16_vk_v6.0.0.bin")
	require.NoError(t, err)
	require.Equal(t, common.HexToHash("0x0e78f4db7a6771a3a6a7d9c3b0de6fe73d58781368967a7fe84d87aefffec896"), common.Hash(sha256.Sum256(vk)))
	raw, err := os.ReadFile("testdata/groth16-fixture-v6.0.0.json")
	require.NoError(t, err)
	var f fixture
	require.NoError(t, json.Unmarshal(raw, &f))
	require.Len(t, f.Proof, ProofLength)
	return Circuit{VK: vk, VKRoot: fixtureVKRoot}, f
}

func TestProductionCircuitConstants(t *testing.T) {
	require.Len(t, CircuitV6_1_0.VK, 492)
	require.Equal(t, common.HexToHash("0x4388a21c687fdd5f218d7e3d13190cac4c5355818d3605fd5fb811df468ee696"), common.Hash(sha256.Sum256(CircuitV6_1_0.VK)))
	require.Equal(t, common.HexToHash("0x002f850ee998974d6cc00e50cd0814b098c05bfade466d28573240d057f25352"), common.Hash(CircuitV6_1_0.VKRoot))
	vk, err := parseVerifyingKey(CircuitV6_1_0.VK)
	require.NoError(t, err)
	require.Len(t, vk.k, PublicInputCount+1)
}

func TestFixtureVerifies(t *testing.T) {
	c, f := loadFixture(t)
	require.Equal(t, [4]byte{0x0e, 0x78, 0xf4, 0xdb}, c.CircuitPrefix())
	require.NoError(t, Verify(c, f.Proof, f.VKey, f.PublicValues))
	// The production circuit refuses the v6.0.0 proof on its prefix.
	require.ErrorIs(t, Verify(CircuitV6_1_0, f.Proof, f.VKey, f.PublicValues), ErrCircuitPrefix)
}

// zeroed returns a copy of b with [from, to) zeroed.
func zeroed(b []byte, from, to int) []byte {
	out := append([]byte(nil), b...)
	clear(out[from:to])
	return out
}

func TestFixtureNegatives(t *testing.T) {
	c, f := loadFixture(t)
	clone := func(b []byte) []byte { return append([]byte(nil), b...) }
	ourVKey := common.HexToHash("0x0011223344556677889900112233445566778899001122334455667788990011")
	cases := []struct {
		name   string
		proof  []byte
		pv     []byte
		vkey   common.Hash
		circ   Circuit
		target error
	}{
		{name: "flipped_proof_byte", proof: func() []byte { p := clone(f.Proof); p[200] ^= 1; return p }()},
		{name: "flipped_last_proof_byte", proof: func() []byte { p := clone(f.Proof); p[ProofLength-1] ^= 1; return p }()},
		{name: "flipped_nonce_byte", proof: func() []byte { p := clone(f.Proof); p[99] ^= 1; return p }(), target: ErrPairing},
		{name: "flipped_public_values_byte", pv: func() []byte { p := clone(f.PublicValues); p[31] ^= 1; return p }(), target: ErrPairing},
		{name: "extra_public_values_byte", pv: append(clone(f.PublicValues), 0), target: ErrPairing},
		{name: "wrong_vkey", vkey: ourVKey, target: ErrPairing},
		{name: "vkey_not_canonical", vkey: common.Hash{0xff}, target: ErrPublicInput},
		{name: "wrong_vk_root_in_proof", proof: func() []byte { p := clone(f.Proof); p[67] ^= 1; return p }(), target: ErrVKRoot},
		{name: "wrong_vk_root_in_circuit", circ: Circuit{VK: c.VK, VKRoot: CircuitV6_1_0.VKRoot}, target: ErrVKRoot},
		{name: "nonzero_exit", proof: func() []byte { p := clone(f.Proof); p[35] = 1; return p }(), target: ErrExitCode},
		{name: "wrong_prefix", proof: func() []byte { p := clone(f.Proof); p[0] ^= 1; return p }(), target: ErrCircuitPrefix},
		{name: "length_355", proof: clone(f.Proof[:ProofLength-1]), target: ErrProofLength},
		{name: "length_357", proof: append(clone(f.Proof), 0), target: ErrProofLength},
		{name: "empty", proof: []byte{}, target: ErrProofLength},
		// Points at infinity (all-zero uncompressed encodings) are rejected before the pairing,
		// as sp1-verifier does (Kona parity: projection/tests.rs groth16::upstream_v6_0_0_fixture).
		{name: "a_at_infinity", proof: zeroed(f.Proof, gnarkProofOffset, gnarkProofOffset+64), target: ErrProofPoint},
		{name: "b_at_infinity", proof: zeroed(f.Proof, gnarkProofOffset+64, gnarkProofOffset+192), target: ErrProofPoint},
		{name: "c_at_infinity", proof: zeroed(f.Proof, gnarkProofOffset+192, gnarkProofOffset+256), target: ErrProofPoint},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			proof, pv, vkey, circ := f.Proof, f.PublicValues, f.VKey, c
			if tc.proof != nil {
				proof = tc.proof
			}
			if tc.pv != nil {
				pv = tc.pv
			}
			if tc.vkey != (common.Hash{}) {
				vkey = tc.vkey
			}
			if tc.circ.VK != nil {
				circ = tc.circ
			}
			err := Verify(circ, proof, vkey, pv)
			require.Error(t, err)
			if tc.target != nil {
				require.ErrorIs(t, err, tc.target)
			}
		})
	}
}

func TestVerifyingKeyParsingIsStrict(t *testing.T) {
	c, f := loadFixture(t)
	bad := append([]byte(nil), c.VK...)
	bad[291] = 5 // nK = 5
	require.ErrorIs(t, Verify(Circuit{VK: bad, VKRoot: c.VKRoot}, append(append([]byte(nil), (Circuit{VK: bad}).prefixBytes()...), f.Proof[4:]...), f.VKey, f.PublicValues), ErrVerifyingKey)
	_, err := parseVerifyingKey(c.VK[:291])
	require.ErrorIs(t, err, ErrVerifyingKey)
	_, err = parseVerifyingKey(c.VK[:292+5*32])
	require.ErrorIs(t, err, ErrVerifyingKey)
}

func (c Circuit) prefixBytes() []byte { p := c.CircuitPrefix(); return p[:] }

func TestPublicValuesDigestMasksTopBits(t *testing.T) {
	d := PublicValuesDigest([]byte("x"))
	full := sha256.Sum256([]byte("x"))
	require.Equal(t, full[0]&0x1f, d[0])
	require.Equal(t, full[1:], d[1:])
}

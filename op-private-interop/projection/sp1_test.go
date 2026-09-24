package projection

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-private-interop/projection/sp1groth16"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func sp1TestConfig(mock bool) Config {
	return Config{Verifier: SP1PrivateProjectionV1, GenesisOutputRoot: common.Hash{9}, ProgramVKey: common.Hash{0x00, 1},
		PrivateConfigHash: common.Hash{3}, DependencySetHash: common.Hash{4}, MockProofs: mock}
}

func mockTestConfig(verifier string) Config {
	return Config{Verifier: verifier, GenesisOutputRoot: common.Hash{9}, DependencySetHash: common.Hash{4}}
}

// The §B.5 gate matrix: {compiled, not compiled} × {901, 902, 10} × {stub, exec-mock, sp1+mock, sp1}.
func TestGateMatrix(t *testing.T) {
	modes := map[string]Config{
		"stub": mockTestConfig(InsecureStub), "exec_mock": mockTestConfig(ExecutionMock),
		"sp1_mock": sp1TestConfig(true), "sp1": sp1TestConfig(false),
	}
	for _, compiled := range []bool{false, true} {
		for _, chain := range []int64{901, 902, 10} {
			for name, cfg := range modes {
				allowed := name == "sp1" || (compiled && chain != 10)
				err := cfg.checkChain(big.NewInt(chain), compiled)
				require.Equal(t, allowed, err == nil, "compiled=%v chain=%d mode=%s err=%v", compiled, chain, name, err)
				v, err := verifierFor(&cfg, big.NewInt(chain), compiled)
				require.Equal(t, allowed, err == nil)
				if name == "sp1" {
					require.Equal(t, SP1Verifier{cfg: cfg, allowMock: false}, v)
				}
			}
		}
	}
	require.False(t, gateAllows(true, nil))
	require.False(t, gateAllows(true, new(big.Int).Lsh(big.NewInt(901), 64)))
	require.True(t, gateAllows(true, big.NewInt(902)))
	require.False(t, gateAllows(false, big.NewInt(901)))
	// A go test binary has the gate compiled in.
	require.True(t, testVerifiersCompiledOrTesting())
	require.NoError(t, (&Config{Verifier: InsecureStub, GenesisOutputRoot: common.Hash{9}, DependencySetHash: common.Hash{4}}).CheckChain(big.NewInt(901)))
}

func TestConfigCheckFieldRules(t *testing.T) {
	require.NoError(t, (&[]Config{sp1TestConfig(false)}[0]).Check())
	var nilCfg *Config
	require.Error(t, nilCfg.Check())
	r := new(big.Int).Set(bn254ScalarModulus)
	for name, change := range map[string]func(*Config){
		"unknown_verifier": func(c *Config) { c.Verifier = "sp1-private-projection-v2" },
		"zero_genesis":     func(c *Config) { c.GenesisOutputRoot = common.Hash{} },
		"zero_vkey":        func(c *Config) { c.ProgramVKey = common.Hash{} },
		"vkey_eq_r":        func(c *Config) { c.ProgramVKey = common.BigToHash(r) },
		"vkey_above_r":     func(c *Config) { c.ProgramVKey = common.Hash{0xff} },
		"zero_private":     func(c *Config) { c.PrivateConfigHash = common.Hash{} },
		"zero_depset":      func(c *Config) { c.DependencySetHash = common.Hash{} },
		"events":           func(c *Config) { c.AllowEvents = true },
	} {
		c := sp1TestConfig(false)
		change(&c)
		require.Error(t, c.Check(), name)
	}
	c := sp1TestConfig(false)
	c.ProgramVKey = common.BigToHash(new(big.Int).Sub(r, big.NewInt(1)))
	require.NoError(t, c.Check(), "r-1 is a valid scalar")
	for _, verifier := range []string{InsecureStub, ExecutionMock} {
		require.NoError(t, (&[]Config{mockTestConfig(verifier)}[0]).Check())
		ev := mockTestConfig(verifier)
		ev.AllowEvents = true
		require.NoError(t, ev.Check(), "the insecure modes may replay events")
		for name, change := range map[string]func(*Config){
			"zero_genesis": func(c *Config) { c.GenesisOutputRoot = common.Hash{} },
			"zero_depset":  func(c *Config) { c.DependencySetHash = common.Hash{} },
			"vkey":         func(c *Config) { c.ProgramVKey = common.Hash{1} },
			"private":      func(c *Config) { c.PrivateConfigHash = common.Hash{1} },
			"mock":         func(c *Config) { c.MockProofs = true },
		} {
			c := mockTestConfig(verifier)
			change(&c)
			require.Error(t, c.Check(), "%s %s", verifier, name)
		}
	}
}

func TestConfigJSONFieldNames(t *testing.T) {
	c := sp1TestConfig(true)
	c.AllowEvents = true
	raw, err := json.Marshal(c)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(raw, &fields))
	for _, k := range []string{"verifier", "genesis_output_root", "program_vkey", "private_config_hash", "dependency_set_hash", "allow_events", "mock_proofs"} {
		require.Contains(t, fields, k)
	}
	require.Len(t, fields, 7)
}

func TestEnvelopeRoundTrip(t *testing.T) {
	var pv [PublicValuesLength]byte
	pv[0], pv[PublicValuesLength-1] = 1, 2
	for _, e := range []Envelope{
		{Kind: EnvelopeKindGroth16, Proof: make([]byte, sp1groth16.ProofLength), PublicValues: pv},
		{Kind: EnvelopeKindMock, Proof: make([]byte, MockProofLength), PublicValues: pv},
	} {
		e.Proof[0] = 7
		raw := EncodeEnvelope(&e)
		require.Equal(t, EnvelopeVersion, raw[0])
		require.Equal(t, e.Kind, raw[1])
		require.Equal(t, len(e.Proof), int(raw[2])<<8|int(raw[3]))
		got, err := DecodeEnvelope(raw)
		require.NoError(t, err)
		require.Equal(t, &e, got)
	}
	require.Len(t, EncodeEnvelope(&Envelope{Kind: EnvelopeKindGroth16, Proof: make([]byte, sp1groth16.ProofLength)}), 1032)
	require.Len(t, MockEnvelope(common.Hash{}, pv), 836)
	require.Nil(t, EncodeEnvelope(&Envelope{Proof: make([]byte, 0x10000)}))
	good := MockEnvelope(common.Hash{1}, pv)
	for name, raw := range map[string][]byte{
		"empty":           nil,
		"header_only":     good[:4],
		"version":         append([]byte{2}, good[1:]...),
		"kind_zero":       append([]byte{1, 0}, good[2:]...),
		"kind_three":      append([]byte{1, 3}, good[2:]...),
		"mock_as_groth16": append([]byte{1, 1}, good[2:]...),
		"mock_len_159":    append([]byte{1, 2, 0, 159}, good[4:len(good)-1]...),
		"trailing":        append(append([]byte(nil), good...), 0),
		"truncated":       good[:len(good)-1],
	} {
		_, err := DecodeEnvelope(raw)
		require.Error(t, err, name)
	}
}

func TestSP1VerifierMock(t *testing.T) {
	cfg := sp1TestConfig(true)
	s := Statement{ChainID: common.BigToHash(big.NewInt(901)), ProjectionHash: common.Hash{1}, OutputsRoot: common.Hash{2}}
	pv := PublicValues(&s)
	env := MockEnvelope(cfg.ProgramVKey, pv)
	require.NoError(t, SP1Verifier{cfg: cfg, allowMock: true}.Verify(s, env))
	require.ErrorContains(t, SP1Verifier{cfg: cfg, allowMock: false}.Verify(s, env), "not enabled")
	other := s
	other.MessagesRoot[0] = 1
	require.ErrorContains(t, SP1Verifier{cfg: cfg, allowMock: true}.Verify(other, env), "at word 19")
	wrong := cfg
	wrong.ProgramVKey[31] ^= 1
	require.ErrorContains(t, SP1Verifier{cfg: wrong, allowMock: true}.Verify(s, env), "program vkey mismatch")
	for i, want := range map[int]string{32: "digest mismatch", 64: "non-zero", 96: "non-zero", 128: "non-zero"} {
		bad := append([]byte(nil), env...)
		bad[4+i+31] ^= 1
		require.ErrorContains(t, SP1Verifier{cfg: cfg, allowMock: true}.Verify(s, bad), want)
	}
}

// The Groth16 path uses the production circuit: the upstream v6.0.0 fixture is refused on its
// circuit prefix, and a garbage proof on its points.
func TestSP1VerifierGroth16Path(t *testing.T) {
	raw, err := os.ReadFile("sp1groth16/testdata/groth16-fixture-v6.0.0.json")
	require.NoError(t, err)
	var f struct {
		VKey  common.Hash   `json:"vkey"`
		Proof hexutil.Bytes `json:"proof"`
	}
	require.NoError(t, json.Unmarshal(raw, &f))
	cfg := sp1TestConfig(false)
	cfg.ProgramVKey = f.VKey
	s := Statement{ChainID: common.BigToHash(big.NewInt(901))}
	env := EncodeEnvelope(&Envelope{Kind: EnvelopeKindGroth16, Proof: f.Proof, PublicValues: PublicValues(&s)})
	require.ErrorIs(t, SP1Verifier{cfg: cfg}.Verify(s, env), sp1groth16.ErrCircuitPrefix)
	proof := make([]byte, sp1groth16.ProofLength)
	prefix := sp1groth16.CircuitV6_1_0.CircuitPrefix()
	copy(proof, prefix[:])
	copy(proof[36:], sp1groth16.CircuitV6_1_0.VKRoot[:])
	env = EncodeEnvelope(&Envelope{Kind: EnvelopeKindGroth16, Proof: proof, PublicValues: PublicValues(&s)})
	err = SP1Verifier{cfg: cfg, allowMock: true}.Verify(s, env)
	require.Error(t, err)
	require.NotErrorIs(t, err, sp1groth16.ErrCircuitPrefix)
}

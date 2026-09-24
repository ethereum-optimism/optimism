package projection_test

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestPublicValuesMagic(t *testing.T) {
	require.Equal(t, crypto.Keccak256Hash([]byte("optimism.private-projection.public-values.v1")), projection.PublicValuesMagic)
	require.Equal(t, 672, projection.PublicValuesLength)
	require.Equal(t, 20, projection.WordTerminalOutput)
}

// Each word lands at its §C.2 position, integers as uint256 big-endian.
func TestPublicValuesLayout(t *testing.T) {
	word := func(i byte) common.Hash { return crypto.Keccak256Hash([]byte{i}) }
	sv := statementVector{
		ChainID: common.BigToHash(big.NewInt(901)), ProjectionConfigHash: word(2), PrivateConfigHash: word(3), DepSetHash: word(4),
		ParentHash: word(5), AnchorNumber: 0x0102030405060708, AnchorHash: word(7), AnchorOutputRoot: word(8), RecoveryHash: word(9),
		FirstBlock: 0x1112131415161718, LastBlock: 0x2122232425262728, ParentOutputRoot: word(12), PrivateTerminalBlockHash: word(13),
		PrivateTerminalParentHash: word(14), L1Head: word(15), PrivateDataHash: word(16), ProjectionHash: word(17),
		OutputsRoot: word(18), MessagesRoot: word(19), TerminalOutput: word(20),
	}
	u256 := func(n uint64) common.Hash { var h common.Hash; binary.BigEndian.PutUint64(h[24:], n); return h }
	want := []common.Hash{
		projection.PublicValuesMagic, sv.ChainID, word(2), word(3), word(4), word(5), u256(sv.AnchorNumber), word(7), word(8), word(9),
		u256(sv.FirstBlock), u256(sv.LastBlock), word(12), word(13), word(14), word(15), word(16), word(17), word(18), word(19), word(20),
	}
	pv := projection.PublicValues(statementFrom(sv))
	for i, w := range want {
		require.Equal(t, w, common.BytesToHash(pv[32*i:32*i+32]), "word %d", i)
	}
	// Claim.Proof and the remaining claim fields are not part of the public values.
	s := statementFrom(sv)
	s.Claim.Proof, s.Claim.RollupConfigHash, s.Claim.AnchorBlock = []byte{1}, common.Hash{1}, 99
	require.Equal(t, pv, projection.PublicValues(s))
}

func TestConfigHashEncoding(t *testing.T) {
	cfg := sp1Config(true)
	cfg.AllowEvents = true
	ctx := context()
	ctx.GenesisNumber, ctx.GenesisTime, ctx.BlockTime = 5, 7, 1
	u64 := func(n uint64) []byte { var b [8]byte; binary.BigEndian.PutUint64(b[:], n); return b[:] }
	chain := common.BigToHash(ctx.ChainID)
	verifier := crypto.Keccak256Hash([]byte(cfg.Verifier))
	want := crypto.Keccak256Hash([]byte("optimism.private-projection-config.v1\x00"), chain[:], u64(5), ctx.GenesisHash[:], u64(7), u64(1),
		verifier[:], cfg.GenesisOutputRoot[:], cfg.ProgramVKey[:], cfg.PrivateConfigHash[:], cfg.DependencySetHash[:], []byte{1}, []byte{1})
	require.Equal(t, want, projection.ConfigHash(&cfg, ctx))
	// Every field and every context input is bound; the continuation and parent are not.
	base := projection.ConfigHash(&cfg, ctx)
	for name, change := range map[string]func(*projection.Config, *projection.Context){
		"verifier":     func(c *projection.Config, _ *projection.Context) { c.Verifier = projection.ExecutionMock },
		"genesis_root": func(c *projection.Config, _ *projection.Context) { c.GenesisOutputRoot[0] ^= 1 },
		"vkey":         func(c *projection.Config, _ *projection.Context) { c.ProgramVKey[31] ^= 1 },
		"private":      func(c *projection.Config, _ *projection.Context) { c.PrivateConfigHash[0] ^= 1 },
		"depset":       func(c *projection.Config, _ *projection.Context) { c.DependencySetHash[0] ^= 1 },
		"events":       func(c *projection.Config, _ *projection.Context) { c.AllowEvents = false },
		"mock":         func(c *projection.Config, _ *projection.Context) { c.MockProofs = false },
		"chain":        func(_ *projection.Config, x *projection.Context) { x.ChainID = big.NewInt(902) },
		"genesis_num":  func(_ *projection.Config, x *projection.Context) { x.GenesisNumber++ },
		"genesis_hash": func(_ *projection.Config, x *projection.Context) { x.GenesisHash[0] ^= 1 },
		"genesis_time": func(_ *projection.Config, x *projection.Context) { x.GenesisTime++ },
		"block_time":   func(_ *projection.Config, x *projection.Context) { x.BlockTime++ },
	} {
		c, x := cfg, ctx
		change(&c, &x)
		require.NotEqual(t, base, projection.ConfigHash(&c, x), name)
	}
	x := ctx
	x.ParentHash, x.L1Head, x.Continuation = common.Hash{0xee}, common.Hash{0xee}, projection.Continuation{}
	require.Equal(t, base, projection.ConfigHash(&cfg, x))
	// A wide chain ID is hashed as its full uint256 word.
	x.ChainID = new(big.Int).Lsh(big.NewInt(1), 200)
	wide := common.BigToHash(x.ChainID)
	require.Equal(t, crypto.Keccak256Hash([]byte(projection.ConfigDomain), wide[:], u64(5), ctx.GenesisHash[:], u64(7), u64(1),
		verifier[:], cfg.GenesisOutputRoot[:], cfg.ProgramVKey[:], cfg.PrivateConfigHash[:], cfg.DependencySetHash[:], []byte{1}, []byte{1}), projection.ConfigHash(&cfg, x))
}

func TestPrivateConfigHash(t *testing.T) {
	a, b := []byte(`{"a":1}`), []byte(`{"b":2}`)
	ha, hb := crypto.Keccak256Hash(a), crypto.Keccak256Hash(b)
	require.Equal(t, crypto.Keccak256Hash([]byte("optimism.private-config.v1\x00"), ha[:], hb[:]), projection.PrivateConfigHash(a, b))
	require.NotEqual(t, projection.PrivateConfigHash(a, b), projection.PrivateConfigHash(b, a))
	// Exact bytes: whitespace changes the hash; nothing is re-serialised.
	require.NotEqual(t, projection.PrivateConfigHash(a, b), projection.PrivateConfigHash([]byte(`{"a": 1}`), b))
}

func TestDependencySetHash(t *testing.T) {
	id := eth.ChainIDFromUInt64
	word := func(n uint64) []byte { var h common.Hash; binary.BigEndian.PutUint64(h[24:], n); return h[:] }
	require.Equal(t, crypto.Keccak256Hash([]byte("optimism.private-dependency-set.v1\x00"), []byte{0, 0, 0, 0, 0, 0, 0, 2}, word(901), word(902)),
		projection.DependencySetHash([]eth.ChainID{id(902), id(901), id(902)}))
	require.Equal(t, crypto.Keccak256Hash([]byte(projection.DepSetDomain), make([]byte, 8)), projection.DependencySetHash(nil))
	require.NotEqual(t, projection.DependencySetHash([]eth.ChainID{id(901)}), projection.DependencySetHash([]eth.ChainID{id(901), id(902)}))
	// Wide IDs sort numerically, after every u64.
	wide := eth.ChainIDFromBig(new(big.Int).Lsh(big.NewInt(1), 200))
	w := wide.Bytes32()
	require.Equal(t, crypto.Keccak256Hash([]byte(projection.DepSetDomain), []byte{0, 0, 0, 0, 0, 0, 0, 2}, word(10), w[:]),
		projection.DependencySetHash([]eth.ChainID{wide, id(10)}))
	// The input is not reordered in place.
	in := []eth.ChainID{id(3), id(1)}
	projection.DependencySetHash(in)
	require.Equal(t, []eth.ChainID{id(3), id(1)}, in)
}

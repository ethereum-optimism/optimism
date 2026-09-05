package genesis

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"
)

func ethSource(t *testing.T) (*core.Genesis, *rollup.Config) {
	t.Helper()
	g := loadPrivateChainGenesis(t)
	deleteStorage(g.Alloc, predeploys.L1BlockAddr, customGasTokenSlot)
	activateProxy(g.Alloc, predeploys.L1BlockAddr, mustBytecode("L1Block"))
	activateProxy(g.Alloc, predeploys.L2ToL1MessagePasserAddr, mustBytecode("L2ToL1MessagePasser"))
	deactivateProxy(g.Alloc, predeploys.NativeAssetLiquidityAddr)
	deactivateProxy(g.Alloc, predeploys.LiquidityControllerAddr)
	cfg := &rollup.Config{L2ChainID: new(big.Int).Set(g.Config.ChainID)}
	cfg.Genesis.L2 = eth.BlockID{Hash: g.ToBlock().Hash(), Number: g.Number}
	cfg.Genesis.L2Time = g.Timestamp
	return g, cfg
}

func TestConfigurePrivateGenesis(t *testing.T) {
	g, cfg := ethSource(t)
	before := genesisJSON(t, g)
	originalHash := cfg.Genesis.L2.Hash
	private, privateCfg, err := ConfigurePrivateGenesis(g, cfg)
	require.NoError(t, err)
	require.Equal(t, before, genesisJSON(t, g))
	require.Equal(t, originalHash, cfg.Genesis.L2.Hash)
	require.NotEqual(t, originalHash, privateCfg.Genesis.L2.Hash)
	require.Equal(t, private.ToBlock().Hash(), privateCfg.Genesis.L2.Hash)
	require.Equal(t, PolicyBridgeCodeHash, implementationCodeHash(private.Alloc, predeploys.SuperchainETHBridgeAddr))
	for _, addr := range []common.Address{predeploys.L1BlockAddr, predeploys.L2StandardBridgeAddr, predeploys.ETHLiquidityAddr} {
		before, after := g.Alloc[addr], private.Alloc[addr]
		require.Equal(t, before.Code, after.Code)
		require.Equal(t, before.Storage, after.Storage)
		require.Equal(t, before.Nonce, after.Nonce)
		if before.Balance != nil {
			require.Zero(t, before.Balance.Cmp(after.Balance))
		}
	}
	projection, err := ProjectGenesisFrom(private)
	require.NoError(t, err)
	// Shared with Rust's policy_profile_matches_the_cross_language_golden_vector.
	require.Equal(t, common.HexToHash("0x452938229ba232178e219a0ebce32533ebfa8654c9ebda74c69b3946151a7395"), private.ToBlock().Hash())
	require.Equal(t, common.HexToHash("0xf460f40066130af21bdaf2fcc3d732572c7e5cf225bc9a306342d75773986e04"), projection.ToBlock().Hash())
	require.Equal(t, common.HexToHash("0xd69dd9061d84611d2868393b68813314b2b01027cf6924ceb85ce872530cf9cc"), projection.ToBlock().Root())
	require.NotEqual(t, private.ToBlock().Hash(), projection.ToBlock().Hash())
	again, _, err := ConfigurePrivateGenesis(private, privateCfg)
	require.NoError(t, err)
	require.Equal(t, private.ToBlock().Hash(), again.ToBlock().Hash())
}

func TestConfigurePrivateGenesisRejectsUnsupportedSource(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*core.Genesis, *rollup.Config)
		want   string
	}{
		{"wrong rollup", func(g *core.Genesis, c *rollup.Config) { c.Genesis.L2.Hash = common.Hash{} }, "does not match"},
		{"CGT", func(g *core.Genesis, c *rollup.Config) {
			a := accountAt(g.Alloc, predeploys.L1BlockAddr)
			a.Storage[customGasTokenSlot] = trueWord
			g.Alloc[predeploys.L1BlockAddr] = a
			c.Genesis.L2.Hash = g.ToBlock().Hash()
		}, "not CGT"},
		{"unknown bridge", func(g *core.Genesis, c *rollup.Config) {
			activateProxy(g.Alloc, predeploys.SuperchainETHBridgeAddr, []byte{0})
			c.Genesis.L2.Hash = g.ToBlock().Hash()
		}, "unsupported source"},
		{"nonempty permissions", func(g *core.Genesis, c *rollup.Config) {
			a := accountAt(g.Alloc, predeploys.SuperchainETHBridgeAddr)
			a.Storage[common.Hash{0xaa}] = trueWord
			g.Alloc[predeploys.SuperchainETHBridgeAddr] = a
			c.Genesis.L2.Hash = g.ToBlock().Hash()
		}, "nonempty policy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, c := ethSource(t)
			tc.mutate(g, c)
			_, _, err := ConfigurePrivateGenesis(g, c)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

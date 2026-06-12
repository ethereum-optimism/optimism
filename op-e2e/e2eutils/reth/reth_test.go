package reth

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestRethInstance boots op-reth against a throwaway L2 genesis and asserts the
// instance satisfies services.EthInstance with working RPCs. It skips when the
// op-reth binary cannot be resolved (no override and no prebuilt binary).
func TestRethInstance(t *testing.T) {
	lgr := testlog.Logger(t, log.LevelInfo)

	if _, err := (rustbin.Spec{SrcDir: "rust", Package: "op-reth", Binary: "op-reth"}).EnsureExists(context.Background(), lgr); err != nil {
		t.Skipf("op-reth binary not available: %v", err)
	}

	genesis := minimalL2Genesis()
	jwtPath := e2eutils.WriteDefaultJWT(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	inst, err := InitL2(ctx, lgr, "reth-test", genesis, jwtPath, Config{})
	require.NoError(t, err, "must boot op-reth")
	t.Cleanup(func() { require.NoError(t, inst.Close()) })

	require.NotEmpty(t, inst.UserRPC().RPC(), "user RPC must resolve")
	require.NotEmpty(t, inst.AuthRPC().RPC(), "auth RPC must resolve")

	client, err := ethclient.DialContext(ctx, endpoint.SelectRPC(endpoint.PreferHttpRPC, inst.UserRPC()))
	require.NoError(t, err)
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	require.NoError(t, err)
	require.Equal(t, genesis.Config.ChainID, chainID, "eth_chainId must match genesis")
}

// minimalL2Genesis builds a self-contained OP-Stack L2 genesis with all hardforks
// through Granite active at genesis. It avoids the contract-artifact-backed
// e2eutils.Setup path so the test stays hermetic (only needs the op-reth binary).
func minimalL2Genesis() *core.Genesis {
	zero := uint64(0)
	cfg := &params.ChainConfig{
		ChainID:                 big.NewInt(0x1a4), // 420
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       big.NewInt(0),
		GrayGlacierBlock:        big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		TerminalTotalDifficulty: big.NewInt(0),
		BedrockBlock:            big.NewInt(0),
		ShanghaiTime:            &zero,
		CancunTime:              &zero,
		RegolithTime:            &zero,
		CanyonTime:              &zero,
		EcotoneTime:             &zero,
		FjordTime:               &zero,
		GraniteTime:             &zero,
		Optimism: &params.OptimismConfig{
			EIP1559Elasticity:  6,
			EIP1559Denominator: 50,
		},
	}
	return &core.Genesis{
		Config:     cfg,
		Nonce:      0,
		Timestamp:  0,
		GasLimit:   30_000_000,
		Difficulty: big.NewInt(0),
		Alloc: types.GenesisAlloc{
			common.HexToAddress("0x1111111111111111111111111111111111111111"): {
				Balance: big.NewInt(1_000_000_000_000_000_000),
			},
		},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
}

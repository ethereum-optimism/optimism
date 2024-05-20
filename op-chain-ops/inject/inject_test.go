package inject

import (
	"math/big"
	"testing"

	opgenesis "github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestInjectState(t *testing.T) {
	genesis := &core.Genesis{
		Config:     &params.ChainConfig{},
		Difficulty: common.Big0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(7),
		Alloc: map[common.Address]types.Account{
			{1}: {
				Balance: big.NewInt(1_000_000_000_000_000_000),
				Nonce:   10,
			},
			{2}: {
				Balance: big.NewInt(100),
				Code:    []byte{5, 7, 8, 3, 4},
				Storage: map[common.Hash]common.Hash{
					{1}: {1},
					{2}: {2},
					{3}: {3},
				},
			},
		},
	}
	db := rawdb.NewDatabase(memorydb.New())
	tdb := triedb.NewDatabase(db, &triedb.Config{Preimages: true})
	genesis.MustCommit(db, tdb)
	hash := rawdb.ReadHeadHeaderHash(db)
	num := rawdb.ReadHeaderNumber(db, hash)
	header := rawdb.ReadHeader(db, hash, *num)
	statedb, err := state.New(header.Root, state.NewDatabaseWithConfig(db, &triedb.Config{Preimages: true}), nil)
	require.NoError(t, err)
	bal := statedb.GetBalance(common.Address{1})
	require.Equal(t, uint256.NewInt(1_000_000_000_000_000_000), bal)

	hardforkTime := uint64(2)
	transitionBlockNumber := int64(1)
	newGenesis := &core.Genesis{
		Config: &params.ChainConfig{
			LondonBlock:                   big.NewInt(transitionBlockNumber),
			ArrowGlacierBlock:             big.NewInt(transitionBlockNumber),
			GrayGlacierBlock:              big.NewInt(transitionBlockNumber),
			MergeNetsplitBlock:            big.NewInt(transitionBlockNumber),
			TerminalTotalDifficulty:       big.NewInt(0),
			TerminalTotalDifficultyPassed: true,
			BedrockBlock:                  big.NewInt(transitionBlockNumber),
			RegolithTime:                  &hardforkTime,
			CanyonTime:                    &hardforkTime,
			ShanghaiTime:                  &hardforkTime,
			CancunTime:                    &hardforkTime,
			EcotoneTime:                   &hardforkTime,
			Optimism: &params.OptimismConfig{
				EIP1559Elasticity:        1,
				EIP1559Denominator:       1,
				EIP1559DenominatorCanyon: 1,
			},
		},
		Difficulty: common.Big1,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(7),
		Alloc: map[common.Address]types.Account{
			{1}: {
				Balance: big.NewInt(1),
				Nonce:   1,
			},
			{2}: {
				Balance: big.NewInt(1),
				Code:    []byte{1},
				Storage: map[common.Hash]common.Hash{
					{1}: {1},
				},
			},
			{3}: {
				Balance: big.NewInt(1),
				Code:    []byte{2},
			},
		},
	}
	err = InjectState(newGenesis, db, &opgenesis.DeployConfig{
		L2OutputOracleStartingBlockNumber: 1,
	}, 0)
	require.NoError(t, err)
	transitionBlock := rawdb.ReadHeadBlock(db)
	statedb, err = state.New(transitionBlock.Header().Root, state.NewDatabaseWithConfig(db, &triedb.Config{Preimages: true}), nil)
	require.NoError(t, err)
	bal = statedb.GetBalance(common.Address{1})
	require.Equal(t, uint256.NewInt(1), bal)
	nonce := statedb.GetNonce(common.Address{1})
	require.Equal(t, uint64(1), nonce)
	bal = statedb.GetBalance(common.Address{2})
	require.Equal(t, uint256.NewInt(1), bal)
	code := statedb.GetCode(common.Address{2})
	require.Equal(t, []byte{1}, code)
	storage := statedb.GetState(common.Address{2}, common.Hash{1})
	require.Equal(t, common.Hash{1}, storage)
	storage = statedb.GetState(common.Address{2}, common.Hash{2})
	require.Equal(t, common.Hash{}, storage)
	bal = statedb.GetBalance(common.Address{3})
	require.Equal(t, uint256.NewInt(1), bal)
	code = statedb.GetCode(common.Address{3})
	require.Equal(t, []byte{2}, code)
	cfg := rawdb.ReadChainConfig(db, hash)
	require.Equal(t, newGenesis.Config, cfg)
}

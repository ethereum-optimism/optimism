package projection_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// The executor parity vector for the projection execution rule (§E): on a public projection,
// every transaction that is not a deposit must execute successfully, or the whole block is
// invalid. User deposits are inert there. op-reth (rust/op-reth/crates/evm/src/projection.rs) and
// Kona (rust/kona/crates/proof/executor/src/builder/core/tests.rs) execute every case in both
// modes and must match the outcome, the error variant and, for valid blocks, the receipt
// statuses, gas used and state root.
//
// Go has no projection execution client. This generator is the reference model: it executes each
// block with op-geth from the env prestate for `execution`, and derives `projection` by skipping
// user deposits (successful, no gas, no state effect) and rejecting the block at the first failed
// non-deposit transaction.
var updateExecutionVectors = flag.Bool("update-execution-vectors", false, "regenerate the shared op-reth/Kona projection executor vectors")

const executionVectorsPath = "testdata/execution.json"

type executionEnv struct {
	ChainID               uint64         `json:"chain_id"`
	Number                uint64         `json:"number"`
	Timestamp             uint64         `json:"timestamp"`
	GasLimit              uint64         `json:"gas_limit"`
	BaseFee               uint64         `json:"base_fee"`
	Coinbase              common.Address `json:"coinbase"`
	PrevRandao            common.Hash    `json:"prev_randao"`
	ParentBeaconBlockRoot common.Hash    `json:"parent_beacon_block_root"`
	Signer                common.Address `json:"signer"`
	// Prestate accounts, all with empty storage. Kona's Isthmus sealing reads the
	// L2ToL1MessagePasser storage root, so that account must exist.
	Prestate map[common.Address]prestateAccount `json:"prestate"`
}

type prestateAccount struct {
	Nonce   uint64        `json:"nonce"`
	Balance *hexutil.Big  `json:"balance"`
	Code    hexutil.Bytes `json:"code"`
}

type executionOutcome struct {
	Valid bool `json:"valid"`
	// Invalid blocks only: the executor error variant and the failing transaction's index.
	Error   string  `json:"error,omitempty"`
	TxIndex *uint64 `json:"tx_index,omitempty"`
	// Valid blocks only.
	Statuses  []uint64     `json:"statuses,omitempty"`
	GasUsed   *uint64      `json:"gas_used,omitempty"`
	StateRoot *common.Hash `json:"state_root,omitempty"`
}

type executionCase struct {
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Transactions []hexutil.Bytes  `json:"transactions"`
	Execution    executionOutcome `json:"execution"`
	Projection   executionOutcome `json:"projection"`
}

type executionVectors struct {
	Description string          `json:"description"`
	Env         executionEnv    `json:"env"`
	Cases       []executionCase `json:"cases"`
}

var executionEnvV1 = executionEnv{
	ChainID:   901,
	Number:    1,
	Timestamp: 2,
	GasLimit:  30_000_000,
	BaseFee:   0,
	Prestate: map[common.Address]prestateAccount{
		predeploys.L2ToL1MessagePasserAddr: {Balance: (*hexutil.Big)(new(big.Int)), Code: []byte{0x00}},
	},
}

var (
	executionKey      = bytes.Repeat([]byte{0x11}, 32)
	revertingInitcode = []byte{0x60, 0x00, 0x60, 0x00, 0xfd} // PUSH1 0; PUSH1 0; REVERT
	loopingInitcode   = []byte{0x5b, 0x60, 0x00, 0x56}       // JUMPDEST; PUSH1 0; JUMP
	callTarget        = common.HexToAddress("0x00000000000000000000000000000000000000bb")
	depositor         = common.HexToAddress("0x00000000000000000000000000000000000000aa")
)

func executionChainConfig() *params.ChainConfig {
	cfg := *params.OptimismTestConfig
	cfg.ChainID = new(big.Int).SetUint64(executionEnvV1.ChainID)
	return &cfg
}

func signedExecutionTx(t *testing.T, nonce uint64, to *common.Address, gas uint64, data []byte) *types.Transaction {
	t.Helper()
	key, err := crypto.ToECDSA(executionKey)
	require.NoError(t, err)
	tx, err := types.SignNewTx(key, types.LatestSignerForChainID(executionChainConfig().ChainID), &types.DynamicFeeTx{
		ChainID:   executionChainConfig().ChainID,
		Nonce:     nonce,
		GasTipCap: new(big.Int),
		GasFeeCap: new(big.Int),
		Gas:       gas,
		To:        to,
		Value:     new(big.Int),
		Data:      data,
	})
	require.NoError(t, err)
	return tx
}

func userDeposit(to *common.Address, data []byte) *types.Transaction {
	return types.NewTx(&types.DepositTx{
		SourceHash: crypto.Keccak256Hash([]byte("optimism.projection-execution-vector.deposit")),
		From:       depositor,
		To:         to,
		Mint:       big.NewInt(1000),
		Value:      big.NewInt(500),
		Gas:        100_000,
		Data:       data,
	})
}

// executeReference runs txs as one block from the env prestate. With projection set it applies
// the projection rule on top of op-geth's execution.
func executeReference(t *testing.T, txs []*types.Transaction, projection bool) executionOutcome {
	t.Helper()
	cfg := executionChainConfig()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	for addr, acc := range executionEnvV1.Prestate {
		statedb.SetNonce(addr, acc.Nonce, tracing.NonceChangeGenesis)
		statedb.SetBalance(addr, uint256.MustFromBig(acc.Balance.ToInt()), tracing.BalanceIncreaseGenesisBalance)
		statedb.SetCode(addr, acc.Code, tracing.CodeChangeGenesis)
	}
	header := &types.Header{
		Number:     new(big.Int).SetUint64(executionEnvV1.Number),
		Time:       executionEnvV1.Timestamp,
		GasLimit:   executionEnvV1.GasLimit,
		BaseFee:    new(big.Int).SetUint64(executionEnvV1.BaseFee),
		Difficulty: new(big.Int),
		Coinbase:   executionEnvV1.Coinbase,
		MixDigest:  executionEnvV1.PrevRandao,
	}
	coinbase := executionEnvV1.Coinbase
	evm := vm.NewEVM(core.NewEVMBlockContext(header, nil, &coinbase, cfg, statedb), statedb, cfg, vm.Config{})
	// The pre-block system calls both Rust clients run. Their contracts are absent, so they must
	// leave no trace in the state root.
	core.ProcessBeaconBlockRoot(executionEnvV1.ParentBeaconBlockRoot, evm)
	core.ProcessParentBlockHash(common.Hash{}, evm)

	gp := core.NewGasPool(header.GasLimit)
	var statuses []uint64
	var gasUsed uint64
	for i, tx := range txs {
		if projection && tx.IsDepositTx() {
			// Inert on the projection: successful, no gas, no state effect.
			statuses = append(statuses, types.ReceiptStatusSuccessful)
			continue
		}
		statedb.SetTxContext(tx.Hash(), i)
		receipt, err := core.ApplyTransaction(evm, gp, statedb, header, tx)
		require.NoError(t, err, "case transaction %d must be valid to include", i)
		if projection && receipt.Status != types.ReceiptStatusSuccessful {
			index := uint64(i)
			return executionOutcome{Valid: false, Error: "ProjectionSequencerTxFailed", TxIndex: &index}
		}
		statuses = append(statuses, receipt.Status)
		gasUsed += receipt.GasUsed
	}
	root := statedb.IntermediateRoot(true)
	return executionOutcome{Valid: true, Statuses: statuses, GasUsed: &gasUsed, StateRoot: &root}
}

func buildExecutionVectors(t *testing.T) executionVectors {
	t.Helper()
	env := executionEnvV1
	key, err := crypto.ToECDSA(executionKey)
	require.NoError(t, err)
	env.Signer = crypto.PubkeyToAddress(key.PublicKey)

	createReverts := signedExecutionTx(t, 0, nil, 100_000, revertingInitcode)
	type spec struct {
		name, description string
		txs               []*types.Transaction
	}
	specs := []spec{
		{"create_reverts", "CREATE whose initcode reverts", []*types.Transaction{createReverts}},
		{"create_oog", "CREATE whose initcode loops until out of gas", []*types.Transaction{
			signedExecutionTx(t, 0, nil, 60_000, loopingInitcode),
		}},
		{"call_empty_ok", "successful plain call with empty calldata", []*types.Transaction{
			signedExecutionTx(t, 0, &callTarget, 21_000, nil),
		}},
		{"deposit_then_revert", "user deposit (inert on the projection) followed by create_reverts", []*types.Transaction{
			userDeposit(&callTarget, nil), createReverts,
		}},
		{"deposit_reverts", "user deposit whose CREATE initcode reverts: never invalidates the block", []*types.Transaction{
			userDeposit(nil, revertingInitcode),
		}},
	}
	vectors := executionVectors{
		Description: "Projection executor parity vectors (spec-sound-profile §E.4). Generated by " +
			"op-private-interop/projection/execution_vectors_test.go with -update-execution-vectors. " +
			"Each case is one block executed from env.prestate under env, with every OP hardfork " +
			"active. `execution` is a non-projection chain; `projection` requires every non-deposit " +
			"transaction to succeed and makes user deposits inert.",
		Env: env,
	}
	for _, s := range specs {
		c := executionCase{Name: s.name, Description: s.description}
		for _, tx := range s.txs {
			raw, err := tx.MarshalBinary()
			require.NoError(t, err)
			c.Transactions = append(c.Transactions, raw)
		}
		c.Execution = executeReference(t, s.txs, false)
		c.Projection = executeReference(t, s.txs, true)
		vectors.Cases = append(vectors.Cases, c)
	}
	return vectors
}

func TestExecutionVectors(t *testing.T) {
	vectors := buildExecutionVectors(t)
	raw, err := json.MarshalIndent(vectors, "", "  ")
	require.NoError(t, err)
	raw = append(raw, '\n')
	if *updateExecutionVectors {
		require.NoError(t, os.WriteFile(executionVectorsPath, raw, 0o644))
	}
	onDisk, err := os.ReadFile(executionVectorsPath)
	require.NoError(t, err)
	require.Equal(t, string(raw), string(onDisk), "run with -update-execution-vectors to regenerate")

	byName := make(map[string]executionCase)
	for _, c := range vectors.Cases {
		byName[c.Name] = c
	}
	// The table of §E.4, pinned independently of the reference model.
	for _, name := range []string{"create_reverts", "create_oog"} {
		c := byName[name]
		require.True(t, c.Execution.Valid, name)
		require.Equal(t, []uint64{0}, c.Execution.Statuses, name)
		require.False(t, c.Projection.Valid, name)
		require.Equal(t, uint64(0), *c.Projection.TxIndex, name)
	}
	ok := byName["call_empty_ok"]
	require.True(t, ok.Execution.Valid)
	require.Equal(t, []uint64{1}, ok.Execution.Statuses)
	require.Equal(t, ok.Execution, ok.Projection)
	mixed := byName["deposit_then_revert"]
	require.True(t, mixed.Execution.Valid)
	require.Equal(t, []uint64{1, 0}, mixed.Execution.Statuses)
	require.False(t, mixed.Projection.Valid)
	require.Equal(t, uint64(1), *mixed.Projection.TxIndex)
	deposit := byName["deposit_reverts"]
	require.True(t, deposit.Execution.Valid)
	require.Equal(t, []uint64{0}, deposit.Execution.Statuses)
	require.True(t, deposit.Projection.Valid)
	require.Equal(t, []uint64{1}, deposit.Projection.Statuses)
	require.Equal(t, uint64(0), *deposit.Projection.GasUsed)
	require.NotEqual(t, *deposit.Execution.StateRoot, *deposit.Projection.StateRoot)
	require.Equal(t, *ok.Projection.StateRoot, *byName["create_reverts"].Execution.StateRoot,
		"a failed create and an empty call both only bump the sender nonce")
}

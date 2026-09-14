package enginetest

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/engineipc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestOpNodeVerifiesRethProofs is the GoSwitch round-2 gate for the account-state read surface, and
// specifically the highest-risk piece of it: historical eth_getProof over the engine's in-memory
// overlay. It proves that op-node's real client stack can:
//
//   - fetch eth_getProof at a HISTORICAL block and cryptographically verify the returned account +
//     storage proof against that block's state root (op-node's TrustRPC=false discipline, and the
//     exact computation L2Client.outputV0 runs to derive the L2 withdrawals/output root); and
//   - read balance/nonce/code/storage at a historical block through the go-ethereum ethclient (the
//     client the switch's L2Engine.EthClient() will hand op-node and the action tests).
//
// A wrong overlay — e.g. answering a historical tag from the tip state — is caught two ways: the
// per-block nonce assertions diverge, and proof.Verify fails against the historical state root.
func TestOpNodeVerifiesRethProofs(t *testing.T) {
	key, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)
	funded := crypto.PubkeyToAddress(key.PublicKey)
	signer := types.LatestSignerForChainID(big.NewInt(chainID))

	genesisPath := writeGenesis(t, funded)

	var logs bytes.Buffer
	proc, err := engineipc.Spawn(enginePath, []string{"--genesis", genesisPath}, &logs, nil)
	require.NoError(t, err, "spawn engine\n%s", logs.String())
	defer func() {
		proc.Close()
		if t.Failed() {
			t.Logf("engine stderr:\n%s", logs.String())
		}
	}()
	ctx := context.Background()
	raw := proc.Client()

	// Build three blocks, each deposit-led plus one user tx from `funded`, so the funded account's
	// nonce equals the block number at every height (block 1 -> nonce 1, ...). Blocks 1 and 2 are
	// then genuinely historical below the tip (block 3), exercising the overlay's per-block state.
	genesisHash := blockHash(t, raw, "earliest")
	b1 := buildBlock(t, raw, genesisHash, 2, []hexutil.Bytes{depositTx(1)}, signTx(t, key, signer, 0))
	b2 := buildBlock(t, raw, b1, 4, []hexutil.Bytes{depositTx(2)}, signTx(t, key, signer, 1))
	buildBlock(t, raw, b2, 6, []hexutil.Bytes{depositTx(3)}, signTx(t, key, signer, 2))

	logger := testlog.Logger(t, gethlog.LevelDebug)
	ethCl, err := sources.NewEthClient(
		client.NewBaseRPCClient(raw), logger, nil, sources.DefaultEthClientConfig(10))
	require.NoError(t, err)

	// eth_getProof at historical blocks: verify the account proof against each block's own state
	// root and confirm the overlay returns that block's state (nonce == block number).
	for _, bn := range []uint64{1, 2} {
		info, err := ethCl.InfoByNumber(ctx, bn)
		require.NoError(t, err)
		proof, err := ethCl.GetProof(ctx, funded, []common.Hash{}, info.Hash().String())
		require.NoError(t, err)
		require.NoError(t, proof.Verify(info.Root()),
			"funded account proof must verify against block %d state root", bn)
		require.EqualValues(t, bn, uint64(proof.Nonce),
			"funded nonce at block %d must reflect that block's historical state", bn)
	}

	// The message-passer proof is the L2 withdrawals/output-root path. This mirrors exactly what
	// L2Client.outputV0 does on the from-state branch: GetProof(L2ToL1MessagePasser, [slot0]) at a
	// block, Verify against the state root, and take the storage hash as the message-passer root —
	// which for an Isthmus block must equal the withdrawalsRoot header field.
	slot0 := common.Hash{}
	info1, err := ethCl.InfoByNumber(ctx, 1)
	require.NoError(t, err)
	mp, err := ethCl.GetProof(ctx, messagePasserAddr, []common.Hash{slot0}, info1.Hash().String())
	require.NoError(t, err)
	require.NoError(t, mp.Verify(info1.Root()), "message-passer proof must verify against state root")
	require.NotNil(t, info1.WithdrawalsRoot(), "Isthmus block carries a withdrawalsRoot header field")
	require.Equal(t, common.Hash(*info1.WithdrawalsRoot()), mp.StorageHash,
		"Isthmus withdrawals root equals the message-passer storage root")
	require.Len(t, mp.StorageProof, 1)
	require.EqualValues(t, 1, bigs.Uint64Strict(mp.StorageProof[0].Value.ToInt()), "seeded message-passer slot 0 == 1")

	// Account reads through the go-ethereum ethclient (the switch's L2Engine.EthClient()): nonce,
	// code, storage, and balance, each served at a historical block.
	gethEth := ethclient.NewClient(raw)

	nonce1, err := gethEth.NonceAt(ctx, funded, big.NewInt(1))
	require.NoError(t, err)
	require.EqualValues(t, 1, nonce1, "eth_getTransactionCount at block 1")

	code, err := gethEth.CodeAt(ctx, codeAddr, big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, seededCode, code, "eth_getCode returns the genesis-seeded bytecode")

	storageVal, err := gethEth.StorageAt(ctx, messagePasserAddr, slot0, big.NewInt(1))
	require.NoError(t, err)
	require.EqualValues(t, 1, bigs.Uint64Strict(new(big.Int).SetBytes(storageVal)), "eth_getStorageAt returns seeded slot 0")

	// Balance is served and strictly decreases as the funded account pays for each block's user tx.
	bal1, err := gethEth.BalanceAt(ctx, funded, big.NewInt(1))
	require.NoError(t, err)
	bal2, err := gethEth.BalanceAt(ctx, funded, big.NewInt(2))
	require.NoError(t, err)
	require.Positive(t, bal1.Cmp(bal2), "funded balance decreases from block 1 to block 2")
	endowment := new(big.Int).SetUint64(1_000_000_000_000_000_000)
	require.Positive(t, endowment.Cmp(bal1), "funded balance is below its genesis endowment after paying gas")
}

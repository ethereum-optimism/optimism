package enginetest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/testengine/chainfixture"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestOpNodeVerifiesRethProofs covers the engine's account-state read surface, and specifically the
// highest-risk piece of it: historical eth_getProof over the engine's in-memory overlay. It proves
// that op-node's real client stack can:
//
//   - fetch eth_getProof at a HISTORICAL block and cryptographically verify the returned account +
//     storage proof against that block's state root (op-node's TrustRPC=false discipline, and the
//     exact computation L2Client.outputV0 runs to derive the L2 withdrawals/output root); and
//   - read balance/nonce/code/storage at a historical block through the go-ethereum ethclient (the
//     client L2Engine.EthClient() hands op-node and the action tests).
//
// A wrong overlay — e.g. answering a historical tag from the tip state — is caught two ways: the
// per-block nonce assertions diverge, and proof.Verify fails against the historical state root.
func TestOpNodeVerifiesRethProofs(t *testing.T) {
	key, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)
	funded := crypto.PubkeyToAddress(key.PublicKey)
	engine := startEngine(t, funded)
	ctx := context.Background()

	// Build three blocks, each deposit-led plus one user tx from `funded`, so the funded account's
	// nonce equals the block number at every height (block 1 -> nonce 1, ...). Blocks 1 and 2 are
	// then genuinely historical below the tip (block 3), exercising the overlay's per-block state.
	genesisHash := chainfixture.BlockHash(t, engine, rpc.EarliestBlockNumber)
	b1 := chainfixture.BuildBlock(t, engine, genesisHash, 2, []hexutil.Bytes{chainfixture.DepositTx(1)}, chainfixture.SignTx(t, key, 0))
	b2 := chainfixture.BuildBlock(t, engine, b1, 4, []hexutil.Bytes{chainfixture.DepositTx(2)}, chainfixture.SignTx(t, key, 1))
	chainfixture.BuildBlock(t, engine, b2, 6, []hexutil.Bytes{chainfixture.DepositTx(3)}, chainfixture.SignTx(t, key, 2))

	logger := testlog.Logger(t, log.LevelDebug)
	ethCl, err := sources.NewEthClient(
		client.NewBaseRPCClient(engine.RPC()), logger, nil, sources.DefaultEthClientConfig(10))
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
	gethEth := ethclient.NewClient(engine.RPC())

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

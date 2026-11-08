package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestL2BatcherModifierDoesNotPoisonCache checks that a block modifier changes only the batcher's
// copy of a block. The client the batcher fetches the block with caches its payload, and the
// other users of that client, such as a sequencer sharing it, must keep reading the block as it is.
func TestL2BatcherModifierDoesNotPoisonCache(gt *testing.T) {
	t := NewDefaultTesting(gt)
	env := SetupEnv(t)
	env.Seq.ActL2PipelineFull(t)
	env.Alice.L2.ActResetTxOpts(t)
	env.Alice.L2.ActMakeTx(t)
	env.Seq.ActL2StartBlock(t)
	env.SeqEngine.ActL2IncludeTx(env.Alice.Address())(t)
	env.Seq.ActL2EndBlock(t)
	head := env.Seq.L2Unsafe()
	// The engine's SourceClient has its own cache, so this is the block as the engine has it.
	want, err := env.SeqEngine.SourceClient(t, 10).PayloadByHash(t.Ctx(), head.Hash)
	require.NoError(t, err)

	engCl := env.SeqEngine.EngineClient(t, env.SetupData.RollupCfg)
	batcher := NewL2Batcher(env.Log, env.SetupData.RollupCfg, DefaultBatcherCfg(env.DeployParams),
		env.Seq.RollupClient(), env.Miner.EthClient(), engCl)
	batcher.ActL2BatchBuffer(t, WithBlockModifier(func(payload *eth.ExecutionPayload) *eth.ExecutionPayload {
		payload.ParentHash = common.MaxHash
		i := PayloadUserTxIndexes(payload)[0]
		tx, err := DecodePayloadTx(t, payload, i).WithSignature(env.Alice.L2.Signer(), make([]byte, 65))
		require.NoError(t, err)
		payload.Transactions[i] = EncodeTx(t, tx)
		return payload
	}))
	require.Equal(t, head, batcher.L2BufferedBlock, "the batcher buffers the block as the engine has it")

	got, err := engCl.PayloadByHash(t.Ctx(), head.Hash)
	require.NoError(t, err)
	require.Equal(t, want.ExecutionPayload, got.ExecutionPayload, "the modifier changed the client's cached payload")
}

func TestPayloadUserTxIndexesSkipsSynthetic(t *testing.T) {
	payload := &eth.ExecutionPayload{Transactions: []eth.Data{
		{optypes.DepositTxType, 0xc0},
		{types.DynamicFeeTxType, 0xc0},
		{0xc0}, // a legacy transaction is an RLP list
		{optypes.PostExecTxType},
	}}
	require.Equal(t, []int{1, 2}, PayloadUserTxIndexes(payload))
}

package msg

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestDepositCannotValidateMessage(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	require := t.Require()
	ctx := t.Ctx()

	sys.L1Network.WaitForOnline()
	alice := sys.FunderL1.NewFundedEOA(eth.OneEther)

	inbox := bindings.NewBindings[bindings.CrossL2Inbox]()
	calldata, err := inbox.ValidateMessage(messages.Identifier{}, eth.Bytes32{}).EncodeInputLambda()
	require.NoError(err, "failed to encode validateMessage calldata")

	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(sys.L2ChainA.DepositContractAddr()),
		bindings.WithTest(t),
	)
	minGas, err := contractio.Read(portal.MinimumGasLimit(uint64(len(calldata))), ctx)
	require.NoError(err, "failed to read minimum deposit gas limit")
	deposit := portal.DepositTransaction(
		predeploys.CrossL2InboxAddr,
		eth.ZeroWei,
		max(100_000, minGas),
		false,
		calldata,
	)
	l1Receipt := contract.Write(alice, deposit)

	var l2Deposit *ethtypes.DepositTx
	for _, log := range l1Receipt.Logs {
		if l2Deposit, err = derive.UnmarshalDepositLogEvent(log); err == nil {
			break
		}
	}
	require.NotNil(l2Deposit, "no TransactionDeposited event in L1 receipt")

	l2Tx := ethtypes.NewTx(l2Deposit)
	sys.L2ELA.WaitL1OriginReached(eth.Unsafe, bigs.Uint64Strict(l1Receipt.BlockNumber), 120)
	l2Receipt := sys.L2ELA.WaitForReceipt(l2Tx.Hash())
	require.Equal(ethtypes.ReceiptStatusFailed, l2Receipt.Status, "deposit unexpectedly validated a message")

	trace := new(wait.TxTrace)
	err = sys.L2ELA.EthClient().RPC().CallContext(ctx, trace, "debug_traceTransaction", hexutil.Bytes(l2Tx.Hash().Bytes()), map[string]any{
		"enableReturnData": true,
		"tracer":           "callTracer",
		"tracerConfig":     map[string]any{},
	})
	require.NoError(err, "failed to trace L2 deposit")
	want := crypto.Keccak256([]byte("NoExecutingDeposits()"))[:4]
	require.Equal(want, []byte(trace.Output), "deposit reverted for an unexpected reason")
}

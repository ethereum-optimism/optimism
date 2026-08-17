package loadtest

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/accounting"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// TODO(16371) every txintent.Call implementation should probably just be a txplan.Option.
func planCall(t devtest.T, call txintent.Call) txplan.Option {
	plan := make([]txplan.Option, 0)
	accessList, err := call.AccessList()
	t.Require().NoError(err)
	if accessList != nil {
		plan = append(plan, txplan.WithAccessList(accessList))
	}
	data, err := call.EncodeInput()
	t.Require().NoError(err)
	if data != nil {
		plan = append(plan, txplan.WithData(data))
	}
	to, err := call.To()
	t.Require().NoError(err)
	if to != nil {
		plan = append(plan, txplan.WithTo(to))
	}
	return txplan.Combine(plan...)
}

func planExecMsg(t devtest.T, initMsg *messages.Message) txplan.Option {
	t.Require().NotNil(initMsg)
	// The EOA plan waits for the chain to reach initMsg: see txintent.WithInteropDependencyWait.
	return planCall(t, &txintent.ExecTrigger{
		Executor: predeploys.CrossL2InboxAddr,
		Msg:      *initMsg,
	})
}

func setupLoadTest(gt *testing.T) (devtest.T, *L2, *L2) {
	if testing.Short() {
		gt.Skip("skipping load test in short mode")
	}
	t := devtest.ParallelT(gt)

	ctx, cancel := context.WithTimeout(t.Ctx(), 3*time.Minute)
	if timeoutStr, exists := os.LookupEnv("NAT_INTEROP_LOADTEST_TIMEOUT"); exists {
		timeout, err := time.ParseDuration(timeoutStr)
		t.Require().NoError(err)
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	t = t.WithCtx(ctx)
	t.Cleanup(cancel)

	l2A, l2B := setupL2s(t)
	collectMetrics(t, l2A.BlockTime)
	return t, l2A, l2B
}

func setupL2s(t devtest.T) (*L2, *L2) {
	sys := presets.NewTwoL2SupernodeInterop(t, 0, presets.WithInteropFilter())
	blockTimeA := time.Duration(sys.L2A.Escape().RollupConfig().BlockTime) * time.Second
	blockTimeB := time.Duration(sys.L2B.Escape().RollupConfig().BlockTime) * time.Second

	l2A := setupL2(t, sys.Wallet, blockTimeA, sys.L2A.Escape().ChainConfig(), sys.L2A.PublicRPC(), sys.FunderA)
	l2B := setupL2(t, sys.Wallet, blockTimeB, sys.L2B.Escape().ChainConfig(), sys.L2B.PublicRPC(), sys.FunderB)

	var deployWg sync.WaitGroup
	defer deployWg.Wait()
	deployWg.Add(1)
	go func() {
		defer deployWg.Done()
		l2A.DeployEventLogger(t)
	}()
	deployWg.Add(1)
	go func() {
		defer deployWg.Done()
		l2B.DeployEventLogger(t)
	}()

	return l2A, l2B
}

func setupL2(t devtest.T, wallet *dsl.HDWallet, blockTime time.Duration, config *params.ChainConfig, el *dsl.L2ELNode, funder *dsl.FunderEOA) *L2 {
	budgetAmount := eth.OneEther
	if budgetStr, exists := os.LookupEnv("NAT_INTEROP_LOADTEST_BUDGET"); exists {
		ethAmt, err := strconv.ParseFloat(budgetStr, 64)
		t.Require().NoError(err)
		weiBig, _ := big.NewFloat(eth.OneEther.WeiFloat() * ethAmt).Int(nil)
		budgetAmount = eth.WeiBig(weiBig)
	}
	const numEOAsPerChain = 1 // TODO(16448): Burst tests exhibit strange behavior with many EOAs.
	amountPerEOA := budgetAmount.Div(numEOAsPerChain)
	innerEOAs := funder.NewFundedEOAs(numEOAsPerChain, amountPerEOA)
	reliableEL := newReliableEL(el.Escape().EthClient(), blockTime, ResubmitterObserver(el.ChainID()))
	eoas := make([]*dsl.SyncEOA, 0, len(innerEOAs))
	budget := accounting.NewBudget(budgetAmount)
	oracle := setupOracle(t, el, blockTime)
	for _, eoa := range innerEOAs {
		p := txinclude.NewPersistent(
			txinclude.NewPkSigner(eoa.Key().Priv(), eoa.ChainID().ToBig()),
			reliableEL,
			txinclude.WithBudget(txinclude.NewTxBudget(budget, txinclude.WithOPCostOracle(oracle))),
		)
		eoas = append(eoas, dsl.NewSyncEOA(p, eoa.Plan()))
	}
	return &L2{
		Config:    config,
		BlockTime: blockTime,
		EOAs:      NewRoundRobin(eoas),
		EL:        el,
		Wallet:    wallet,
	}
}

func collectMetrics(t devtest.T, blockTime time.Duration) {
	metricsCollector := NewMetricsCollector(blockTime)
	ctx, cancel := context.WithCancel(t.Ctx())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.Require().NoError(metricsCollector.Start(ctx))
	}()
	t.Cleanup(func() {
		cancel()
		wg.Wait()

		dir := filepath.Join("artifacts", t.Name()+"_"+time.Now().Format("20060102-150405"))
		t.Require().NoError(os.MkdirAll(dir, 0755))
		t.Require().NoError(metricsCollector.SaveGraphs(dir))
	})
}

func setupOracle(t devtest.T, el *dsl.L2ELNode, blockTime time.Duration) *txinclude.CostOracle {
	oracle := txinclude.NewCostOracle(&batchRPCClient{
		multicaller: el.Escape().EthClient().NewMultiCaller(3),
	}, blockTime)
	t.Require().NoError(oracle.SetParams(t.Ctx()))

	ctx, cancel := context.WithCancel(t.Ctx())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		oracle.Start(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	return oracle
}

type reliableEL struct {
	*txinclude.Resubmitter
	*txinclude.Monitor
}

var _ txinclude.EL = (*reliableEL)(nil)

func newReliableEL(el txinclude.EL, blockTime time.Duration, observer txinclude.ResubmitterObserver) *reliableEL {
	return &reliableEL{
		Resubmitter: txinclude.NewResubmitter(el, blockTime, txinclude.WithObserver(observer)),
		Monitor:     txinclude.NewMonitor(el, blockTime),
	}
}

// initMsgFromReceipt turns the first log in the receipt into an inititiating message.
func initMsgFromReceipt(t devtest.T, l2 *L2, receipt *optypes.Receipt) (*messages.Message, error) {
	ref, err := l2.EL.Escape().EthClient().BlockRefByHash(t.Ctx(), receipt.BlockHash)
	if err != nil {
		return nil, fmt.Errorf("get init msg block ref by hash: %w", err)
	}
	out := new(txintent.InteropOutput)
	if err := out.FromReceipt(t.Ctx(), &receipt.Receipt, ref, l2.EL.ChainID()); err != nil {
		return nil, fmt.Errorf("get init msg from receipt: %w", err)
	}
	t.Require().NotEmpty(out.Entries)
	return &out.Entries[0], nil
}

type batchRPCClient struct {
	multicaller *batching.MultiCaller
}

var _ txinclude.RPCClient = (*batchRPCClient)(nil)

func (b *batchRPCClient) BatchCallContext(ctx context.Context, elems []rpc.BatchElem) error {
	calls := make([]batching.Call, 0, len(elems))
	for _, elem := range elems {
		calls = append(calls, &batchCall{
			elem: elem,
		})
	}
	// rpcblock.Latest won't be used, see that ToBatchElemCreator below ignores the block.
	_, err := b.multicaller.Call(ctx, rpcblock.Latest, calls...)
	return err
}

type batchCall struct {
	elem rpc.BatchElem
}

var _ batching.Call = (*batchCall)(nil)

func (b *batchCall) ToBatchElemCreator() (batching.BatchElementCreator, error) {
	return func(_ rpcblock.Block) (any, rpc.BatchElem) {
		return b.elem.Result, b.elem
	}, nil
}

func (b *batchCall) HandleResult(result any) (*batching.CallResult, error) {
	val, ok := result.(*hexutil.Bytes)
	if !ok {
		return nil, fmt.Errorf("response %v was not a *hexutil.Bytes", result)
	}
	return batching.NewCallResult([]any{val}), nil
}

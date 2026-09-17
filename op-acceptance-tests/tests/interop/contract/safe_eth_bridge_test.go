package contract

import (
	"crypto/ecdsa"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Keep the experimental bindings local; deploy the actual Foundry artifact, not copied bytecode.
type safeETHTransfer struct {
	Sender    common.Address
	Recipient common.Address
	Amount    *big.Int
	Nonce     *big.Int
	Deadline  *big.Int
}

type safeETHRecord struct {
	Transfer safeETHTransfer
	Status   uint8
}

type safeETHBindings struct {
	SendETH           func(common.Address, *big.Int) bindings.TypedCall[[32]byte] `sol:"sendETH"`
	InitiateTransfer  func(common.Address, *big.Int) bindings.TypedCall[[32]byte] `sol:"initiateTransfer"`
	AbortTransfer     func([32]byte) bindings.TypedCall[any]                      `sol:"abortTransfer"`
	Source            func([32]byte) bindings.TypedCall[safeETHRecord]            `sol:"source"`
	Destination       func([32]byte) bindings.TypedCall[safeETHRecord]            `sol:"destination"`
	ReservedLiquidity func() bindings.TypedCall[*big.Int]                         `sol:"reservedLiquidity"`
}

type safeETHFixture struct {
	t                                     devtest.T
	sys                                   *presets.TwoL2SupernodeInterop
	sender, relayerA, relayerB, recipient *dsl.EOA
	bridge                                common.Address
	initialLiquidityA, initialLiquidityB  *big.Int
	a, b                                  safeETHBindings
}

func newSafeETHFixture(t devtest.T) *safeETHFixture {
	owners := make(map[eth.ChainID]*ecdsa.PrivateKey)
	sys := presets.NewTwoL2SupernodeInterop(t, 0, presets.WithDeployerOptions(
		func(t devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
			for _, chain := range builder.L2s() {
				key, err := keys.Secret(devkeys.ChainOperatorKey{ChainID: chain.ChainID().ToBig(), Role: devkeys.L2ProxyAdminOwnerRole})
				t.Require().NoError(err)
				owners[chain.ChainID()] = key
			}
		},
	))
	f := &safeETHFixture{t: t, sys: sys}
	f.sender = sys.FunderA.NewFundedEOA(eth.OneEther)
	f.relayerA = sys.FunderA.NewFundedEOA(eth.OneEther)
	f.relayerB = sys.FunderB.NewFundedEOA(eth.OneEther)
	f.recipient = sys.Wallet.NewEOA(sys.L2ELB)

	// Opt-in replacement of the existing bridge via the devnet's real proxy admin.
	// No messenger/inbox code or network upgrade bundles are modified.
	deployerA := dsl.NewEOA(dsl.NewKey(t, owners[sys.L2ELA.ChainID()]), sys.L2ELA)
	deployerB := dsl.NewEOA(dsl.NewKey(t, owners[sys.L2ELB.ChainID()]), sys.L2ELB)
	f.relayerA.Transfer(deployerA.Address(), eth.OneTenthEther)
	f.relayerB.Transfer(deployerB.Address(), eth.OneTenthEther)
	f.bridge = predeploys.SuperchainETHBridgeAddr
	wd, err := os.Getwd()
	t.Require().NoError(err)
	root, err := opservice.FindMonorepoRoot(wd)
	t.Require().NoError(err)
	artifact, err := foundry.ReadArtifact(filepath.Join(root, "packages/contracts-bedrock/forge-artifacts/SafeETHBridge.sol/SafeETHBridge.json"))
	t.Require().NoError(err, "build contracts before running this test")
	args, err := artifact.ABI.Pack("", deployerA.ChainID().ToBig(), deployerB.ChainID().ToBig())
	t.Require().NoError(err)
	code := append(append([]byte{}, artifact.Bytecode.Object...), args...)
	for _, deployer := range []*dsl.EOA{deployerA, deployerB} {
		tx := deployer.Transact(deployer.Plan(), txplan.WithData(code))
		receipt := tx.Included.Value()
		t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
		admin := bindings.NewBindings[struct {
			Upgrade func(common.Address, common.Address) bindings.TypedCall[any] `sol:"upgrade"`
		}](bindings.WithTo(predeploys.ProxyAdminAddr))
		upgrade, err := contractio.Plan(admin.Upgrade(f.bridge, receipt.ContractAddress))
		t.Require().NoError(err)
		deployer.Transact(deployer.Plan(), upgrade)
	}
	f.a = bindings.NewBindings[safeETHBindings](bindings.WithTo(f.bridge), bindings.WithClient(sys.L2ELA.EthClient()))
	f.b = bindings.NewBindings[safeETHBindings](bindings.WithTo(f.bridge), bindings.WithClient(sys.L2ELB.EthClient()))
	f.initialLiquidityA, err = sys.L2ELA.EthClient().BalanceAt(t.Ctx(), predeploys.ETHLiquidityAddr, nil)
	t.Require().NoError(err)
	f.initialLiquidityB, err = sys.L2ELB.EthClient().BalanceAt(t.Ctx(), predeploys.ETHLiquidityAddr, nil)
	t.Require().NoError(err)
	return f
}

func (f *safeETHFixture) initiate(deadline uint64) (*txintent.IntentTx[*bindings.TypedCall[[32]byte], *txintent.InteropOutput], [32]byte) {
	f.t.Logger().Info("Escrowing native ETH", "bridge", f.bridge, "deadline", deadline)
	tx := txintent.NewIntent[*bindings.TypedCall[[32]byte], *txintent.InteropOutput](f.sender.Plan(), txplan.WithValue(eth.OneHundredthEther))
	call := f.a.InitiateTransfer(f.recipient.Address(), new(big.Int).SetUint64(deadline))
	tx.Content.Set(&call)
	receipt, err := tx.PlannedTx.Included.Eval(f.t.Ctx())
	f.t.Require().NoError(err)
	f.t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
	// TransferPrepared is emitted last and indexes the protocol transfer ID.
	id := [32]byte(receipt.Logs[len(receipt.Logs)-1].Topics[1])
	return tx, id
}

func relaySafeETH[C txintent.Call](f *safeETHFixture, actor *dsl.EOA, origin *dsl.L2ELNode, previous *txintent.IntentTx[C, *txintent.InteropOutput]) *txintent.IntentTx[*txintent.RelayTrigger, *txintent.InteropOutput] {
	f.t.Logger().Info("Relaying bridge message", "destination", actor.ChainID())
	receipt := previous.PlannedTx.Included.Value()
	f.sys.Supernode.AwaitValidatedTimestamp(origin.BlockRefByHash(receipt.BlockHash).Time)
	index := -1
	for i, entry := range receipt.Logs {
		if entry.Address == predeploys.L2toL2CrossDomainMessengerAddr && len(entry.Topics) == 4 && entry.Topics[0] == crypto.Keccak256Hash([]byte("SentMessage(uint256,address,uint256,address,bytes)")) {
			index = i
			break
		}
	}
	f.t.Require().NotEqual(-1, index, "expected an outgoing messenger event")
	tx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](actor.Plan())
	tx.Content.DependOn(&previous.Result)
	tx.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &previous.Result, &previous.PlannedTx.Included, index))
	relayed, err := tx.PlannedTx.Included.Eval(f.t.Ctx())
	f.t.Require().NoError(err, "bridge message must relay")
	f.t.Require().Equal(types.ReceiptStatusSuccessful, relayed.Status)
	return tx
}

func (f *safeETHFixture) verify(id [32]byte, source, destination uint8, sourceLiquidity, destinationLiquidity eth.ETH) {
	a, err := contractio.Read(f.a.Source(id), f.t.Ctx())
	f.t.Require().NoError(err)
	b, err := contractio.Read(f.b.Destination(id), f.t.Ctx())
	f.t.Require().NoError(err)
	f.t.Require().Equal(source, a.Status, "source decision")
	f.t.Require().Equal(destination, b.Status, "destination state")
	la, err := contractio.Read(f.a.ReservedLiquidity(), f.t.Ctx())
	f.t.Require().NoError(err)
	lb, err := contractio.Read(f.b.ReservedLiquidity(), f.t.Ctx())
	f.t.Require().NoError(err)
	f.t.Require().Equal(sourceLiquidity, eth.WeiBig(la), "source reserved mint capacity")
	f.t.Require().Equal(destinationLiquidity, eth.WeiBig(lb), "destination reserved mint capacity")
	poolA, err := f.sys.L2ELA.EthClient().BalanceAt(f.t.Ctx(), predeploys.ETHLiquidityAddr, nil)
	f.t.Require().NoError(err)
	poolB, err := f.sys.L2ELB.EthClient().BalanceAt(f.t.Ctx(), predeploys.ETHLiquidityAddr, nil)
	f.t.Require().NoError(err)
	bridgeA, err := f.sys.L2ELA.EthClient().BalanceAt(f.t.Ctx(), f.bridge, nil)
	f.t.Require().NoError(err)
	bridgeB, err := f.sys.L2ELB.EthClient().BalanceAt(f.t.Ctx(), f.bridge, nil)
	f.t.Require().NoError(err)
	expectedA, expectedB := new(big.Int).Set(f.initialLiquidityA), new(big.Int).Set(f.initialLiquidityB)
	if source == 2 {
		expectedA.Add(expectedA, a.Transfer.Amount)
	}
	if destination == 2 {
		expectedB.Sub(expectedB, a.Transfer.Amount)
	}
	f.t.Require().Zero(poolA.Cmp(expectedA), "source escrow only locks into ETHLiquidity at commit")
	f.t.Require().Zero(poolB.Cmp(expectedB), "destination mints only at commit")
	expectedEscrow := new(big.Int)
	if source == 1 {
		expectedEscrow.Set(a.Transfer.Amount)
	}
	f.t.Require().Zero(bridgeA.Cmp(expectedEscrow), "source escrow balance")
	f.t.Require().Zero(bridgeB.Sign(), "PREPARE must not mint provisional destination ETH")
}

func TestSafeETHBridgePaysOnlyAfterSourceCommit(gt *testing.T) {
	f := newSafeETHFixture(devtest.ParallelT(gt))
	before := f.sender.GetBalance()
	sent, id := f.initiate(f.sys.L2ELA.BlockRefByLabel(eth.Unsafe).Time + 600)
	f.sender.VerifyBalanceLessThan(before.Sub(eth.OneHundredthEther)) // Includes transaction fees.
	f.recipient.VerifyBalanceExact(eth.ZeroWei)
	f.verify(id, 1, 0, eth.ZeroWei, eth.ZeroWei)

	prepared := relaySafeETH(f, f.relayerB, f.sys.L2ELA, sent)
	reserved := eth.OneHundredthEther
	f.verify(id, 1, 1, eth.ZeroWei, reserved)
	f.recipient.VerifyBalanceExact(eth.ZeroWei)

	committed := relaySafeETH(f, f.relayerA, f.sys.L2ELB, prepared)
	f.verify(id, 2, 1, eth.ZeroWei, reserved)
	f.recipient.VerifyBalanceExact(eth.ZeroWei)

	relaySafeETH(f, f.relayerB, f.sys.L2ELA, committed)
	f.verify(id, 2, 2, eth.ZeroWei, eth.ZeroWei)
	f.recipient.VerifyBalanceExact(eth.OneHundredthEther)
}

func TestSafeETHBridgeRefundPreventsLateAcknowledgement(gt *testing.T) {
	f := newSafeETHFixture(devtest.ParallelT(gt))
	deadline := f.sys.L2ELA.BlockRefByLabel(eth.Unsafe).Time + 30
	sent, id := f.initiate(deadline)
	prepared := relaySafeETH(f, f.relayerB, f.sys.L2ELA, sent)
	f.verify(id, 1, 1, eth.ZeroWei, eth.OneHundredthEther)
	f.sys.L2ELA.WaitForUnsafe(func(info eth.BlockInfo) (bool, error) { return info.Time() >= deadline, nil })
	before := f.sender.GetBalance()
	aborted := txintent.NewIntent[*bindings.TypedCall[any], *txintent.InteropOutput](f.sender.Plan())
	call := f.a.AbortTransfer(id)
	aborted.Content.Set(&call)
	receipt, err := aborted.PlannedTx.Included.Eval(f.t.Ctx())
	f.t.Require().NoError(err)
	f.t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
	// Refund minus ordinary transaction fees still increases the sender's balance.
	f.t.Require().True(f.sender.GetBalance().Gt(before), "sender must receive refund")
	f.recipient.VerifyBalanceExact(eth.ZeroWei)

	relaySafeETH(f, f.relayerA, f.sys.L2ELB, prepared) // Late ACK must succeed without committing.
	f.verify(id, 3, 1, eth.ZeroWei, eth.OneHundredthEther)
	relaySafeETH(f, f.relayerB, f.sys.L2ELA, aborted)
	f.verify(id, 3, 3, eth.ZeroWei, eth.ZeroWei)
	f.recipient.VerifyBalanceExact(eth.ZeroWei)
}

// Safe bridging is optional: ordinary sendETH still uses one relay and pays native ETH,
// including while the same destination has a pending safe transfer.
func TestSafeETHBridgeOrdinaryTransferAlongsideSafe(gt *testing.T) {
	f := newSafeETHFixture(devtest.ParallelT(gt))
	safe, id := f.initiate(f.sys.L2ELA.BlockRefByLabel(eth.Unsafe).Time + 600)
	prepared := relaySafeETH(f, f.relayerB, f.sys.L2ELA, safe)
	f.verify(id, 1, 1, eth.ZeroWei, eth.OneHundredthEther)
	f.recipient.VerifyBalanceExact(eth.ZeroWei)

	ordinary := txintent.NewIntent[*bindings.TypedCall[[32]byte], *txintent.InteropOutput](f.sender.Plan(), txplan.WithValue(eth.OneHundredthEther))
	call := f.a.SendETH(f.recipient.Address(), f.sys.L2ELB.ChainID().ToBig())
	ordinary.Content.Set(&call)
	receipt, err := ordinary.PlannedTx.Included.Eval(f.t.Ctx())
	f.t.Require().NoError(err)
	f.t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
	// Account for ordinary immediate locking separately from the safe transfer.
	f.initialLiquidityA.Add(f.initialLiquidityA, eth.OneHundredthEther.ToBig())
	f.verify(id, 1, 1, eth.ZeroWei, eth.OneHundredthEther)
	relaySafeETH(f, f.relayerB, f.sys.L2ELA, ordinary)
	f.initialLiquidityB.Sub(f.initialLiquidityB, eth.OneHundredthEther.ToBig())
	f.verify(id, 1, 1, eth.ZeroWei, eth.OneHundredthEther)
	f.recipient.VerifyBalanceExact(eth.OneHundredthEther)

	committed := relaySafeETH(f, f.relayerA, f.sys.L2ELB, prepared)
	f.verify(id, 2, 1, eth.ZeroWei, eth.OneHundredthEther)
	relaySafeETH(f, f.relayerB, f.sys.L2ELA, committed)
	f.verify(id, 2, 2, eth.ZeroWei, eth.ZeroWei)
	f.recipient.VerifyBalanceExact(eth.OneHundredthEther.Add(eth.OneHundredthEther))
}

package privateinterop

import (
	"context"
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type batcherSystemConfig struct {
	SetBatcherHash func(common.Hash) bindings.TypedCall[any] `sol:"setBatcherHash"`
}

// Rotation may expire unpublished blocks from the old batcher. The replacement
// must eventually publish from the recovered private tip using only its new key.
func TestPrivatePublicationResumesAfterBatcherRotation(gt *testing.T) {
	t := devtest.SerialT(gt)
	configs := make(map[eth.ChainID]*bss.CLIConfig)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(10)),
		presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck()),
		presets.WithBatcherOption(func(target sysgo.ComponentTarget, cfg *bss.CLIConfig) { configs[target.ChainID] = cfg }))
	sys.L2BCL.Advanced(safety.CrossSafe, 24, 120)
	sys.L2BatcherB.Stop()
	next := sys.FunderL1.NewFundedEOA(eth.OneEther)
	ownerKey := sys.L2B.Escape().Keys().Secret(devkeys.SystemConfigOwner.Key(sys.L2B.ChainID().ToBig()))
	owner := dsl.NewKey(t, ownerKey).User(sys.L1EL)
	sys.FunderL1.FundAtLeast(owner, eth.OneEther)
	config := bindings.NewBindings[batcherSystemConfig](bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(sys.L2B.Escape().Deployment().SystemConfigProxyAddr()), bindings.WithTest(t))
	_, err := contractio.Write(config.SetBatcherHash(eth.AddressAsLeftPaddedHash(next.Address())), t.Ctx(), owner.Plan())
	t.Require().NoError(err)
	attrs := bindings.NewBindings[bindings.L1Block](bindings.WithClient(sys.L2ELB.EthClient()), bindings.WithTo(predeploys.L1BlockAddr), bindings.WithTest(t))
	t.Require().Eventually(func() bool {
		got, err := contractio.Read(attrs.BatcherHash(), t.Ctx())
		return err == nil && got == eth.AddressAsLeftPaddedHash(next.Address())
	}, time.Minute, time.Second, "private derivation must observe the new batcher")
	changedAt := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	replacementConfig := *configs[sys.L2B.ChainID()]
	replacementConfig.TxMgrConfig.PrivateKey = common.Bytes2Hex(crypto.FromECDSA(next.Key().Priv()))
	replacementConfig.TxMgrConfig.Mnemonic = ""
	replacementConfig.RPC.ListenPort = 0
	service, err := bss.BatcherServiceFromCLIConfig(t.Ctx(), func(err error) { t.Errorf("replacement batcher stopped: %v", err) }, "rotation-test", &replacementConfig, t.Logger().New("component", "replacement-batcher"))
	t.Require().NoError(err)
	t.Cleanup(func() { ctx, cancel := context.WithCancel(t.Ctx()); cancel(); _ = service.Stop(ctx) })
	t.Require().NoError(service.Start(t.Ctx()))
	sys.L2BCL.Reached(safety.CrossSafe, changedAt.Number+24, 240)
	alice := sys.FunderB.NewFundedEOA(eth.OneEther)
	transfer := alice.Transfer(common.Address{0xeb}, eth.OneGWei)
	rec, err := transfer.Included.Eval(t.Ctx())
	t.Require().NoError(err)
	sys.L2BCL.ReachedRef(safety.CrossSafe, eth.BlockID{Hash: rec.BlockHash, Number: bigs.Uint64Strict(rec.BlockNumber)}, 180)
}

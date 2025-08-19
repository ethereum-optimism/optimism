package dsl

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
)

type MinBaseFee struct {
	commonImpl

	l1Client     *L1ELNode
	l2Network    *L2Network
	l2EL         *L2ELNode
	systemConfig minBaseFeeSystemConfig

	originalSignificand uint8
	originalExponent    uint8
}

type minBaseFeeSystemConfig struct {
	SetMinBaseFee         func(sig uint8, exp uint8) bindings.TypedCall[any] `sol:"setMinBaseFee"`
	MinBaseFeeSignificand func() bindings.TypedCall[uint8]                   `sol:"minBaseFeeSignificand"`
	MinBaseFeeExponent    func() bindings.TypedCall[uint8]                   `sol:"minBaseFeeExponent"`
}

func NewMinBaseFee(t devtest.T, l2Network *L2Network, l1EL *L1ELNode, l2EL *L2ELNode) *MinBaseFee {
	systemConfig := bindings.NewBindings[minBaseFeeSystemConfig](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(l2Network.Escape().Deployment().SystemConfigProxyAddr()),
		bindings.WithTest(t))

	originalSig, err := contractio.Read(systemConfig.MinBaseFeeSignificand(), t.Ctx())
	t.Require().NoError(err, "reading original minBaseFeeSignificand")
	originalExp, err := contractio.Read(systemConfig.MinBaseFeeExponent(), t.Ctx())
	t.Require().NoError(err, "reading original minBaseFeeExponent")

	return &MinBaseFee{
		commonImpl:          commonFromT(t),
		l1Client:            l1EL,
		l2Network:           l2Network,
		l2EL:                l2EL,
		systemConfig:        systemConfig,
		originalSignificand: originalSig,
		originalExponent:    originalExp,
	}
}

func (mbf *MinBaseFee) CheckCompatibility() bool {
	_, err := contractio.Read(mbf.systemConfig.MinBaseFeeSignificand(), mbf.ctx)
	if err != nil {
		mbf.t.Skipf("MinBaseFee methods not available in devstack: %v", err)
		return false
	}
	return true
}

func (mbf *MinBaseFee) GetSystemOwner() *EOA {
	priv := mbf.l2Network.Escape().Keys().Secret(devkeys.SystemConfigOwner.Key(mbf.l2Network.ChainID().ToBig()))
	return NewKey(mbf.t, priv).User(mbf.l1Client)
}

func (mbf *MinBaseFee) SetMinBaseFeeFactors(significand, exponent uint8) {
	owner := mbf.GetSystemOwner()

	_, err := contractio.Write(mbf.systemConfig.SetMinBaseFee(significand, exponent), mbf.ctx, owner.Plan())
	mbf.require.NoError(err, "SetMinBaseFee transaction failed")

	mbf.t.Logf("Set min base fee factors on L1: significand=%d, exponent=%d", significand, exponent)
}

func (mbf *MinBaseFee) WaitForL2Sync(expectedSignificand, expectedExponent uint8) {
	expected := eip1559.EncodeMinBaseFeeFactors(expectedSignificand, expectedExponent)
	mbf.waitForMinBaseFeeFactors(expected)
}

func (mbf *MinBaseFee) VerifyL2Config(expectedSignificand, expectedExponent uint8) {
	expected := eip1559.EncodeMinBaseFeeFactors(expectedSignificand, expectedExponent)
	client := mbf.l2EL.Escape().L2EthClient()
	ext, ok := client.(apis.L2EthExtendedClient)
	mbf.require.True(ok, "L2 client does not support extended payload API")

	payload, err := ext.PayloadByLabel(mbf.ctx, "latest")
	mbf.require.NoError(err, "failed to get latest payload")
	mbf.require.True(len(payload.ExecutionPayload.ExtraData) == 10, "payload extra data should be 10 bytes")

	got := uint8(payload.ExecutionPayload.ExtraData[9])
	mbf.require.Equal(expected, got, "L2 min base fee factors do not match expected")
}

func (mbf *MinBaseFee) CheckBaseFeeCanDecrease() {
	// Ensure we are past genesis and collect a small sample across advancing blocks
	_ = mbf.l2EL.WaitForBlock()
	el := mbf.l2EL.Escape().EthClient()
	bases := make([]*big.Int, 0, 6)
	info, err := el.InfoByLabel(mbf.ctx, "latest")
	mbf.require.NoError(err)
	bases = append(bases, info.BaseFee())
	for i := 0; i < 5; i++ {
		_ = mbf.l2EL.WaitForBlock()
		next, err := el.InfoByLabel(mbf.ctx, "latest")
		mbf.require.NoError(err)
		bases = append(bases, next.BaseFee())
	}
	decreased := false
	for i := 1; i < len(bases); i++ {
		if bases[i].Cmp(bases[i-1]) < 0 {
			decreased = true
			break
		}
	}
	mbf.require.True(decreased, "expected base-fee to decrease when minBaseFee=0")
}

func (mbf *MinBaseFee) VerifyMinBaseFeeClamp(minBase *big.Int) {
	// Give the sequencer one more block, then check 5 consecutive blocks
	_ = mbf.l2EL.WaitForBlock()
	el := mbf.l2EL.Escape().EthClient()

	// Check 5 consecutive blocks to ensure min base fee is consistently applied
	for i := 1; i <= 5; i++ {
		_ = mbf.l2EL.WaitForBlock()
		info, err := el.InfoByLabel(mbf.ctx, "latest")
		mbf.require.NoError(err)
		mbf.require.True(info.BaseFee().Cmp(minBase) >= 0, "block %d base-fee %s should be >= %s", info.NumberU64(), info.BaseFee(), minBase)
	}
}

func (mbf *MinBaseFee) RestoreOriginalConfig() {
	mbf.SetMinBaseFeeFactors(mbf.originalSignificand, mbf.originalExponent)
	mbf.WaitForL2Sync(mbf.originalSignificand, mbf.originalExponent)
}

// waitForMinBaseFeeFactors waits until the L2 latest payload extra-data encodes the expected factors.
func (mbf *MinBaseFee) waitForMinBaseFeeFactors(expected uint8) {
	client := mbf.l2EL.Escape().L2EthClient()
	ext, ok := client.(apis.L2EthExtendedClient)
	mbf.require.True(ok, "L2 client does not support extended payload API")

	mbf.require.Eventually(func() bool {
		payload, err := ext.PayloadByLabel(mbf.ctx, "latest")
		if err != nil {
			return false
		}
		if len(payload.ExecutionPayload.ExtraData) != 10 {
			return false
		}
		got := uint8(payload.ExecutionPayload.ExtraData[9])
		return got == expected
	}, 2*time.Minute, 5*time.Second, "L2 min base fee factors did not sync within timeout")
}

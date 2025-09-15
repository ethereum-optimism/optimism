package custom_gas_token

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// TestCGT_MessengerRejectsValue ensures that sending native value to the
// L2CrossDomainMessenger reverts under CGT (non-payable path).
func TestCGT_MessengerRejectsValue(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	ensureCGTOrSkip(t, sys)

	ctx, cancel := context.WithTimeout(t.Ctx(), 30*time.Second)
	defer cancel()

	from := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther).Address()
	_, err := sys.L2EL.Escape().L2EthClient().EstimateGas(ctx, ethereum.CallMsg{
		From:  from,
		To:    &l2XDMAddr,
		Value: big.NewInt(1), // 1 wei native
		Data:  nil,
	})
	if err == nil {
		t.Require().Fail("expected estimation error when sending value to L2CrossDomainMessenger in CGT mode")
	}
}

// TestCGT_L2StandardBridge_LegacyWithdrawReverts verifies that the legacy
// ETH-specific withdraw path on L2StandardBridge reverts under CGT.
func TestCGT_L2StandardBridge_LegacyWithdrawReverts(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	ensureCGTOrSkip(t, sys)

	ctx, cancel := context.WithTimeout(t.Ctx(), 30*time.Second)
	defer cancel()

	bridgeABIJSON := `[
		{"inputs":[
			{"internalType":"address","name":"_l2Token","type":"address"},
			{"internalType":"uint256","name":"_amount","type":"uint256"},
			{"internalType":"uint32","name":"_minGasLimit","type":"uint32"},
			{"internalType":"bytes","name":"_extraData","type":"bytes"}
		],
		"name":"withdraw","outputs":[],"stateMutability":"payable","type":"function"}
	]`
	bridgeABI, err := abi.JSON(strings.NewReader(bridgeABIJSON))
	if err != nil {
		t.Require().Fail("%v", err)
	}
	// Any address is fine; the ETH-specific legacy path should be disabled under CGT.
	anyAddress := l2XDMAddr
	data, err := bridgeABI.Pack("withdraw", anyAddress, big.NewInt(1), uint32(100_000), []byte{})
	if err != nil {
		t.Require().Fail("%v", err)
	}

	from := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther).Address()
	_, err = sys.L2EL.Escape().L2EthClient().EstimateGas(ctx, ethereum.CallMsg{
		From: from,
		To:   &l2BridgeAddr,
		Data: data,
	})
	if err == nil {
		t.Require().Fail("expected estimation error for L2StandardBridge.withdraw under CGT")
	}
}

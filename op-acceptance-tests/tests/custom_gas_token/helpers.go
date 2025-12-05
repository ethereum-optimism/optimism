// op-acceptance-tests/tests/custom_gas_token/helpers.go
package custom_gas_token

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
)

var (
	// L2 predeploy: L1Block (address is stable across OP Stack chains)
	l1BlockAddr = common.HexToAddress("0x4200000000000000000000000000000000000015")

	// L2 predeploy: L2CrossDomainMessenger & L2StandardBridge (for revert checks)
	l2XDMAddr    = common.HexToAddress("0x4200000000000000000000000000000000000007")
	l2BridgeAddr = common.HexToAddress("0x4200000000000000000000000000000000000010")
)

// isCGTEnabled checks if CGT mode is enabled without skipping the test.
// Returns true if CGT is enabled, false if native ETH mode, and false if the check fails.
func isCGTEnabled(t devtest.T, sys *presets.Minimal) bool {
	l2 := sys.L2EL.Escape().L2EthClient()
	isCustomGasTokenFunc := w3.MustNewFunc("isCustomGasToken()", "bool")

	ctx, cancel := context.WithTimeout(t.Ctx(), 20*time.Second)
	defer cancel()

	data, _ := isCustomGasTokenFunc.EncodeArgs()
	out, err := l2.Call(ctx, ethereum.CallMsg{To: &l1BlockAddr, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		return false
	}

	var isCustom bool
	if err := isCustomGasTokenFunc.DecodeReturns(out, &isCustom); err != nil {
		return false
	}

	return isCustom
}

// getCGTMetadata retrieves the name and symbol of the custom gas token.
// Returns empty strings if CGT is not enabled or if the call fails.
func getCGTMetadata(t devtest.T, sys *presets.Minimal) (string, string) {
	l2 := sys.L2EL.Escape().L2EthClient()
	gasPayingTokenNameFunc := w3.MustNewFunc("gasPayingTokenName()", "string")
	gasPayingTokenSymbolFunc := w3.MustNewFunc("gasPayingTokenSymbol()", "string")

	ctx, cancel := context.WithTimeout(t.Ctx(), 20*time.Second)
	defer cancel()

	// Read name
	data, _ := gasPayingTokenNameFunc.EncodeArgs()
	out, err := l2.Call(ctx, ethereum.CallMsg{To: &l1BlockAddr, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		return "", ""
	}
	var name string
	if err := gasPayingTokenNameFunc.DecodeReturns(out, &name); err != nil {
		return "", ""
	}

	// Read symbol
	data, _ = gasPayingTokenSymbolFunc.EncodeArgs()
	out, err = l2.Call(ctx, ethereum.CallMsg{To: &l1BlockAddr, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		return "", ""
	}
	var symbol string
	if err := gasPayingTokenSymbolFunc.DecodeReturns(out, &symbol); err != nil {
		return "", ""
	}

	return name, symbol
}

// ensureCGTOrSkip probes L2 L1Block for CGT mode. If not enabled, the test is skipped.
// Returns (name, symbol).
func ensureCGTOrSkip(t devtest.T, sys *presets.Minimal) (string, string) {
	if !isCGTEnabled(t, sys) {
		t.Skip("CGT disabled on this devnet (native ETH mode detected)")
	}

	name, symbol := getCGTMetadata(t, sys)
	if name == "" || symbol == "" {
		t.Skip("Failed to retrieve CGT metadata")
	}

	return name, symbol
}

// SkipIfCGT probes L2 L1Block for CGT mode. If CGT is enabled, the test is skipped.
// This is useful for tests that should only run with native ETH.
func SkipIfCGT(t devtest.T, sys *presets.Minimal) {
	if isCGTEnabled(t, sys) {
		t.Skip("Test skipped: CGT is enabled (test requires native ETH)")
	}
}

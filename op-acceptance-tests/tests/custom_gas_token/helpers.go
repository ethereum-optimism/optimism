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

var (
	fnIsCustomGasToken     = w3.MustNewFunc("isCustomGasToken()", "bool")
	fnGasPayingTokenName   = w3.MustNewFunc("gasPayingTokenName()", "string")
	fnGasPayingTokenSymbol = w3.MustNewFunc("gasPayingTokenSymbol()", "string")
)

// ensureCGTOrSkip probes L2 L1Block for CGT mode. If not enabled, the test is skipped.
// Returns (name, symbol).
func ensureCGTOrSkip(t devtest.T, sys *presets.Minimal) (string, string) {
	l2 := sys.L2EL.Escape().L2EthClient()

	ctx, cancel := context.WithTimeout(t.Ctx(), 20*time.Second)
	defer cancel()

	// isCustomGasToken()
	data, _ := fnIsCustomGasToken.EncodeArgs()
	out, err := l2.Call(ctx, ethereum.CallMsg{To: &l1BlockAddr, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		t.Skipf("CGT not enabled (isCustomGasToken() call failed): %v", err)
	}
	var isCGT bool
	t.Require().NoError(fnIsCustomGasToken.DecodeReturns(out, &isCGT))
	if !isCGT {
		t.Skip("CGT disabled on this devnet (native ETH mode detected)")
	}

	// Read metadata (name/symbol)
	data, _ = fnGasPayingTokenName.EncodeArgs()
	out, err = l2.Call(ctx, ethereum.CallMsg{To: &l1BlockAddr, Data: data}, rpc.LatestBlockNumber)
	t.Require().NoError(err)
	var name string
	t.Require().NoError(fnGasPayingTokenName.DecodeReturns(out, &name))

	data, _ = fnGasPayingTokenSymbol.EncodeArgs()
	out, err = l2.Call(ctx, ethereum.CallMsg{To: &l1BlockAddr, Data: data}, rpc.LatestBlockNumber)
	t.Require().NoError(err)
	var symbol string
	t.Require().NoError(fnGasPayingTokenSymbol.DecodeReturns(out, &symbol))

	return name, symbol
}

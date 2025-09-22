// op-acceptance-tests/tests/custom_gas_token/helpers.go
package custom_gas_token

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	// L2 predeploy: L1Block (address is stable across OP Stack chains)
	l1BlockAddr = common.HexToAddress("0x4200000000000000000000000000000000000015")

	// L2 predeploy: L2CrossDomainMessenger & L2StandardBridge (for revert checks)
	l2XDMAddr    = common.HexToAddress("0x4200000000000000000000000000000000000007")
	l2BridgeAddr = common.HexToAddress("0x4200000000000000000000000000000000000010")
)

const igasTokenABI = `[
  {"inputs":[],"name":"isCustomGasToken","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"gasPayingTokenName","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"gasPayingTokenSymbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"}
]`

// ensureCGTOrSkip probes L2 L1Block for CGT mode. If not enabled, the test is skipped.
// Returns (name, symbol).
func ensureCGTOrSkip(t devtest.T, sys *presets.Minimal) (string, string) {
	l2 := sys.L2EL.Escape().L2EthClient()

	abiGT, err := abi.JSON(strings.NewReader(igasTokenABI))
	t.Require().NoError(err)

	ctx, cancel := context.WithTimeout(t.Ctx(), 20*time.Second)
	defer cancel()

	// isCustomGasToken()
	data, _ := abiGT.Pack("isCustomGasToken")
	out, err := l2.Call(ctx, ethereum.CallMsg{To: &l1BlockAddr, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		t.Skipf("CGT not enabled (isCustomGasToken() call failed): %v", err)
	}
	vals, err := abiGT.Unpack("isCustomGasToken", out)
	t.Require().NoError(err)
	if !vals[0].(bool) {
		t.Skip("CGT disabled on this devnet (native ETH mode detected)")
	}

	// Read metadata (name/symbol)
	data, _ = abiGT.Pack("gasPayingTokenName")
	out, err = l2.Call(ctx, ethereum.CallMsg{To: &l1BlockAddr, Data: data}, rpc.LatestBlockNumber)
	t.Require().NoError(err)
	vn, err := abiGT.Unpack("gasPayingTokenName", out)
	t.Require().NoError(err)
	name := vn[0].(string)

	data, _ = abiGT.Pack("gasPayingTokenSymbol")
	out, err = l2.Call(ctx, ethereum.CallMsg{To: &l1BlockAddr, Data: data}, rpc.LatestBlockNumber)
	t.Require().NoError(err)
	vs, err := abiGT.Unpack("gasPayingTokenSymbol", out)
	t.Require().NoError(err)
	symbol := vs[0].(string)

	return name, symbol
}

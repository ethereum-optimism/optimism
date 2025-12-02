package custom_gas_token

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestMain(m *testing.M) {
	// Create a CGT-enabled devnet with 1M tokens of liquidity
	liq := new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(1e18)) // 1M tokens * 18 decimals

	presets.DoMain(m,
		presets.WithMinimal(),
		stack.MakeCommon(sysgo.WithDeployerOptions(
			sysgo.WithCustomGasToken("Custom Gas Token", "CGT", liq, common.Address{}),
		)),
	)
}

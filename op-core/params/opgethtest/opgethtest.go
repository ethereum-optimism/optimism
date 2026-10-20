// Package opgethtest builds op-geth chain configs for differential tests of the OP Stack
// chain config and genesis against op-geth.
package opgethtest

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
)

// ChainConfig returns op-geth's OP Stack test chain config with chain ID chainID and the
// fork schedule s, built independently of op-core/params. Shanghai, Cancun, Prague and
// Osaka activate with Canyon, Ecotone, Isthmus and Karst. The EIP-1559 parameters are
// op-geth's test defaults: elasticity 6, denominator 50, Canyon denominator 250. Like the
// configs op-chain-ops writes, it has no consensus engine config.
func ChainConfig(t testing.TB, chainID uint64, s forkstest.Schedule) *params.ChainConfig {
	cfg := *params.OptimismTestConfig
	cfg.ChainID = new(big.Int).SetUint64(chainID)
	// OptimismTestConfig inherits an ethash config from go-ethereum's merged test config.
	cfg.Ethash = nil
	fields := reflect.ValueOf(&cfg).Elem()
	for _, fork := range forkstest.TimestampELForks {
		name := forkstest.Title(fork) + "Time"
		field := fields.FieldByName(name)
		require.Truef(t, field.IsValid(), "op-geth ChainConfig lacks field %s", name)
		field.Set(reflect.ValueOf(s(fork)))
	}
	cfg.ShanghaiTime = s(forks.Canyon)
	cfg.CancunTime = s(forks.Ecotone)
	cfg.PragueTime = s(forks.Isthmus)
	cfg.OsakaTime = s(forks.Karst)
	return &cfg
}

package op_e2e

import (
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
)

func TestGasPriceOracle(t *testing.T) {
	backend := backends.NewSimulatedBackend(map[common.Address]core.GenesisAccount{
		predeploys.GasPriceOracleAddr: {
			Code:    common.FromHex(bindings.GasPriceOracleDeployedBin),
			Balance: big.NewInt(0),
			Storage: map[common.Hash]common.Hash{
				common.HexToHash("0x0"): common.HexToHash("0x0101"), // isEcotone = true, isFjord = true
			},
		},
		predeploys.L1BlockAddr: {
			Code:    common.FromHex(bindings.L1BlockDeployedBin),
			Balance: big.NewInt(0),
		},
	}, math.MaxUint64)

	caller, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, backend)
	assert.NoError(t, err)

	atLeastOnce := false
	err = filepath.WalkDir("../specs", func(path string, d fs.DirEntry, err error) error {
		atLeastOnce = true

		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		used, err := caller.GetL1GasUsed(&bind.CallOpts{}, b)
		if err != nil {
			return err
		}

		expected := (types.FlzCompressLen(b) + 68) * 16
		assert.Equal(t, used.Uint64(), uint64(expected), path)

		return nil
	})
	assert.NoError(t, err)
	assert.True(t, atLeastOnce)
}

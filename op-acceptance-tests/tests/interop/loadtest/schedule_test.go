package loadtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/eip1559"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

func TestL2GasTarget(t *testing.T) {
	t.Parallel()
	const (
		holoceneTime = 100
		jovianTime   = 200
		gasLimit     = 60_000_000
	)
	forks := &opparams.ChainConfig{
		ChainID:      big.NewInt(901),
		Optimism:     &opparams.OptimismConfig{EIP1559Elasticity: 6, EIP1559Denominator: 50},
		HoloceneTime: ptr.New(uint64(holoceneTime)),
		JovianTime:   ptr.New(uint64(jovianTime)),
	}
	block := func(time uint64, extra []byte) eth.BlockInfo {
		return eth.HeaderBlockInfo(&types.Header{Number: big.NewInt(1), Time: time, GasLimit: gasLimit, Extra: extra})
	}

	t.Run("holocene", func(t *testing.T) {
		target, err := l2GasTarget(forks, block(holoceneTime, eip1559.EncodeHoloceneExtraData(250, 4)))
		require.NoError(t, err)
		require.Equal(t, uint64(gasLimit/4), target)
	})
	t.Run("jovian", func(t *testing.T) {
		target, err := l2GasTarget(forks, block(jovianTime, eip1559.EncodeJovianExtraData(250, 3, 1000)))
		require.NoError(t, err)
		require.Equal(t, uint64(gasLimit/3), target)
	})
	t.Run("pre-holocene", func(t *testing.T) {
		_, err := l2GasTarget(forks, block(holoceneTime-1, nil))
		require.ErrorContains(t, err, "predates Holocene")
	})
	t.Run("invalid extraData", func(t *testing.T) {
		_, err := l2GasTarget(forks, block(jovianTime, nil))
		require.ErrorContains(t, err, "invalid EIP-1559 extraData")
	})
	t.Run("zero elasticity", func(t *testing.T) {
		_, err := l2GasTarget(forks, block(holoceneTime, eip1559.EncodeHoloceneExtraData(250, 0)))
		require.ErrorContains(t, err, "invalid EIP-1559 extraData")
	})
	t.Run("holocene extraData after jovian", func(t *testing.T) {
		_, err := l2GasTarget(forks, block(jovianTime, eip1559.EncodeHoloceneExtraData(250, 4)))
		require.ErrorContains(t, err, "invalid EIP-1559 extraData")
	})
}

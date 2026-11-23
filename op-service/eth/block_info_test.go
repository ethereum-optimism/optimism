package eth

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestBlobBaseFee(t *testing.T) {
	cfg := params.MainnetChainConfig
	cancun := *cfg.CancunTime
	ebg := uint64(10_000_000)
	header := func(time uint64, excessBlobGas *uint64) *types.Header {
		return &types.Header{Number: big.NewInt(1), Time: time, ExcessBlobGas: excessBlobGas}
	}
	infos := map[string]func(*types.Header) BlockInfo{
		"header": HeaderBlockInfo,
		"block":  func(h *types.Header) BlockInfo { return BlockToInfo(types.NewBlockWithHeader(h)) },
	}
	for name, info := range infos {
		t.Run(name, func(t *testing.T) {
			t.Run("no excess blob gas", func(t *testing.T) {
				fee, err := info(header(cancun, nil)).BlobBaseFee(cfg)
				require.NoError(t, err)
				require.Nil(t, fee)
			})
			t.Run("blob fork active", func(t *testing.T) {
				fee, err := info(header(cancun, &ebg)).BlobBaseFee(cfg)
				require.NoError(t, err)
				require.Equal(t, CalcBlobFeeCancun(ebg), fee)
			})
			t.Run("excess blob gas above maximum", func(t *testing.T) {
				huge := uint64(math.MaxUint64)
				done := make(chan error, 1)
				go func() {
					_, err := info(header(cancun, &huge)).BlobBaseFee(cfg)
					done <- err
				}()
				select {
				case err := <-done:
					require.Error(t, err)
				case <-time.After(5 * time.Second):
					t.Fatal("BlobBaseFee did not return")
				}
			})
			t.Run("blob fee exponent bound", func(t *testing.T) {
				// A large update fraction keeps a large excess blob gas valid.
				const fraction = 30_000_000
				sched := *cfg.BlobScheduleConfig
				sched.Cancun = &params.BlobConfig{Target: 3, Max: 6, UpdateFraction: fraction}
				bigFraction := *cfg
				bigFraction.BlobScheduleConfig = &sched

				valid := uint64(1<<32 + 1)
				fee, err := info(header(cancun, &valid)).BlobBaseFee(&bigFraction)
				require.NoError(t, err)
				require.NotNil(t, fee)

				belowBound := uint64(178*fraction - 1)
				fee, err = info(header(cancun, &belowBound)).BlobBaseFee(&bigFraction)
				require.NoError(t, err)
				require.NotNil(t, fee)

				atBound := uint64(178 * fraction)
				_, err = info(header(cancun, &atBound)).BlobBaseFee(&bigFraction)
				require.Error(t, err)
			})
			t.Run("excess blob gas before blob fork", func(t *testing.T) {
				_, err := info(header(cancun-1, &ebg)).BlobBaseFee(cfg)
				require.Error(t, err)
			})
		})
	}
}

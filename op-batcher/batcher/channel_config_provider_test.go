package batcher

import (
	"context"
	"errors"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"
)

type mockGasPricer struct {
	err         error
	tipCap      int64
	baseFee     int64
	blobTipCap  int64
	blobBaseFee int64
}

func (gp *mockGasPricer) SuggestGasPriceCaps(context.Context) (tipCap *big.Int, baseFee *big.Int, blobTipCap *big.Int, blobBaseFee *big.Int, err error) {
	if gp.err != nil {
		return nil, nil, nil, nil, gp.err
	}
	return big.NewInt(gp.tipCap), big.NewInt(gp.baseFee), big.NewInt(gp.blobTipCap), big.NewInt(gp.blobBaseFee), nil
}

func TestDynamicEthChannelConfig_ChannelConfig(t *testing.T) {
	calldataCfg := ChannelConfig{
		MaxFrameSize:    120_000 - 1,
		TargetNumFrames: 1,
	}
	blobCfg := ChannelConfig{
		MaxFrameSize:    eth.MaxBlobDataSize - 1,
		TargetNumFrames: 3, // gets closest to amortized fixed tx costs
		UseBlobs:        true,
	}

	tests := []struct {
		name         string
		tipCap       int64
		baseFee      int64
		blobBaseFee  int64
		isAmsterdam  bool
		wantCalldata bool
		isThrottling bool
	}{
		{
			name:        "much-cheaper-blobs",
			tipCap:      1e3,
			baseFee:     1e6,
			blobBaseFee: 1,
		},
		{
			name:        "close-cheaper-blobs-before-amsterdam",
			tipCap:      1e3,
			baseFee:     1e6,
			blobBaseFee: 398e5, // this value is just under the pre-Amsterdam equilibrium point for 3 blobs
		},
		{
			name:         "close-cheaper-calldata-before-amsterdam",
			tipCap:       1e3,
			baseFee:      1e6,
			blobBaseFee:  399e5, // this value is just over the pre-Amsterdam equilibrium point for 3 blobs
			wantCalldata: true,
		},
		{
			name:        "close-cheaper-blobs-after-amsterdam",
			tipCap:      1e3,
			baseFee:     1e6,
			blobBaseFee: 636e5, // this value is just under the Amsterdam equilibrium point for 3 blobs
			isAmsterdam: true,
		},
		{
			name:         "close-cheaper-calldata-after-amsterdam",
			tipCap:       1e3,
			baseFee:      1e6,
			blobBaseFee:  637e5, // this value is just over the Amsterdam equilibrium point for 3 blobs
			isAmsterdam:  true,
			wantCalldata: true,
		},
		{
			name:         "much-cheaper-calldata",
			tipCap:       1e3,
			baseFee:      1e6,
			blobBaseFee:  1e9,
			wantCalldata: true,
		},
		{
			// blobs should be chosen even though calldata is cheaper.
			name:         "throttling-is-enabled",
			tipCap:       1e3,
			baseFee:      1e6,
			blobBaseFee:  1e9,
			isThrottling: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lgr, ch := testlog.CaptureLogger(t, slog.LevelInfo)
			gp := &mockGasPricer{
				tipCap:      tt.tipCap,
				baseFee:     tt.baseFee,
				blobBaseFee: tt.blobBaseFee,
			}
			dec := NewDynamicEthChannelConfig(lgr, 1*time.Second, gp, blobCfg, calldataCfg)
			cc := dec.ChannelConfig(tt.isThrottling, tt.isAmsterdam)
			if tt.wantCalldata {
				require.Equal(t, cc, calldataCfg)
				require.NotNil(t, ch.FindLog(testlog.NewMessageContainsFilter("calldata")))
				require.Same(t, &dec.calldataConfig, dec.lastConfig)
			} else {
				require.Equal(t, cc, blobCfg)
				require.NotNil(t, ch.FindLog(testlog.NewMessageContainsFilter("blob")))
				require.Same(t, &dec.blobConfig, dec.lastConfig)
			}
		})
	}

	t.Run("error-latest", func(t *testing.T) {
		lgr, ch := testlog.CaptureLogger(t, slog.LevelInfo)
		gp := &mockGasPricer{
			tipCap:      1,
			baseFee:     1e3,
			blobBaseFee: 1e6, // should return calldata cfg without error
			err:         errors.New("gp-error"),
		}
		dec := NewDynamicEthChannelConfig(lgr, 1*time.Second, gp, blobCfg, calldataCfg)
		require.Equal(t, dec.ChannelConfig(false, false), blobCfg)
		require.NotNil(t, ch.FindLog(
			testlog.NewLevelFilter(slog.LevelWarn),
			testlog.NewMessageContainsFilter("returning last config"),
		))

		gp.err = nil
		require.Equal(t, dec.ChannelConfig(false, false), calldataCfg)
		require.NotNil(t, ch.FindLog(
			testlog.NewLevelFilter(slog.LevelInfo),
			testlog.NewMessageContainsFilter("calldata"),
		))

		gp.err = errors.New("gp-error-2")
		require.Equal(t, dec.ChannelConfig(false, false), calldataCfg)
		require.NotNil(t, ch.FindLog(
			testlog.NewLevelFilter(slog.LevelWarn),
			testlog.NewMessageContainsFilter("returning last config"),
		))
	})
}

func TestComputeSingleCalldataTxCost(t *testing.T) {
	tests := []struct {
		name        string
		isAmsterdam bool
		want        *big.Int
	}{
		{name: "pectra", want: big.NewInt(2_442_000)},                       // (21_000 + 40*30_000) * (1+1)
		{name: "amsterdam", isAmsterdam: true, want: big.NewInt(3_870_000)}, // (15_000 + 64*30_000) * (1+1)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeSingleCalldataTxCost(30_000, big.NewInt(1), big.NewInt(1), tt.isAmsterdam)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestComputeSingleBlobTxCost(t *testing.T) {
	tests := []struct {
		name        string
		isAmsterdam bool
		want        *big.Int
	}{
		{name: "pectra", want: big.NewInt(21_013_520)},                       // 21_000 * (1+1) + 131_072*5*32
		{name: "amsterdam", isAmsterdam: true, want: big.NewInt(21_001_520)}, // 15_000 * (1+1) + 131_072*5*32
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeSingleBlobTxCost(5, big.NewInt(1), big.NewInt(1), big.NewInt(32), tt.isAmsterdam)
			require.Equal(t, tt.want, got)
		})
	}
}

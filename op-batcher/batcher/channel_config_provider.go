package batcher

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const randomByteCalldataGas = params.TxDataNonZeroGasEIP2028

type (
	ChannelConfigProvider interface {
		ChannelConfigFull() ChannelConfig
		ChannelConfig(txData) ChannelConfig
	}

	GasPricer interface {
		SuggestGasPriceCaps(ctx context.Context) (tipCap *big.Int, baseFee *big.Int, blobBaseFee *big.Int, err error)
	}

	DynamicEthChannelConfig struct {
		log       log.Logger
		timeout   time.Duration // query timeout
		gasPricer GasPricer

		blobConfig     ChannelConfig
		calldataConfig ChannelConfig
		lastConfig     *ChannelConfig
	}
)

func NewDynamicEthChannelConfig(lgr log.Logger,
	reqTimeout time.Duration, gasPricer GasPricer,
	blobConfig ChannelConfig, calldataConfig ChannelConfig,
) *DynamicEthChannelConfig {
	dec := &DynamicEthChannelConfig{
		log:            lgr,
		timeout:        reqTimeout,
		gasPricer:      gasPricer,
		blobConfig:     blobConfig,
		calldataConfig: calldataConfig,
	}
	// start with blob config
	dec.lastConfig = &dec.blobConfig
	return dec
}

// ChannelConfig will perform a detailed comparison of costs to submit the supplied
// txData either as a single calldata transaction or as a single blob transaction
// taking into account current market conditions. It returns a ChannelConfig
// of the appropriate type depending on which DA type is cheaper.
func (dec *DynamicEthChannelConfig) ChannelConfig(d txData) ChannelConfig {
	ctx, cancel := context.WithTimeout(context.Background(), dec.timeout)
	defer cancel()
	tipCap, baseFee, blobBaseFee, err := dec.gasPricer.SuggestGasPriceCaps(ctx)
	if err != nil {
		dec.log.Warn("Error querying gas prices, returning last config", "err", err)
		return *dec.lastConfig
	}

	// Calldata Calculation
	callDataBytes := d.CallData()
	callDataGas, err := core.IntrinsicGas(callDataBytes, nil, false, true, true, true)
	callDataGas += params.TxGas
	calldataPrice := new(big.Int).Add(baseFee, tipCap)
	calldataCost := new(big.Int).Mul(big.NewInt(int64(callDataGas)), calldataPrice)

	// Blob Calculation
	blobs, err := d.Blobs()
	blobGas := big.NewInt(params.BlobTxBlobGasPerBlob * int64(len(blobs)))
	blobCost := new(big.Int).Mul(blobGas, blobBaseFee)
	// blobs still have intrinsic calldata costs
	blobCalldataCost := new(big.Int).Mul(big.NewInt(int64(params.TxGas)), calldataPrice)
	blobCost = new(big.Int).Add(blobCost, blobCalldataCost)

	lgr := dec.log.New("base_fee", baseFee, "blob_base_fee", blobBaseFee, "tip_cap", tipCap,
		"calldata_cost", calldataCost,
		"blob_cost", blobCost)

	// Comparison
	if blobCost.Cmp(calldataCost) > 0 {
		lgr.Info("Using calldata channel config")
		dec.lastConfig = &dec.calldataConfig
		return dec.calldataConfig
	}
	lgr.Info("Using blob channel config")
	dec.lastConfig = &dec.blobConfig
	return dec.blobConfig

}
func (dec *DynamicEthChannelConfig) ChannelConfigFull() ChannelConfig {
	ctx, cancel := context.WithTimeout(context.Background(), dec.timeout)
	defer cancel()
	tipCap, baseFee, blobBaseFee, err := dec.gasPricer.SuggestGasPriceCaps(ctx)
	if err != nil {
		dec.log.Warn("Error querying gas prices, returning last config", "err", err)
		return *dec.lastConfig
	}

	// We estimate the gas costs of a calldata and blob tx under the assumption that we'd fill
	// a frame fully and compressed random channel data has few zeros, so they can be
	// ignored in the calldata gas price estimation.
	// It is also assumed that a calldata tx would contain exactly one full frame
	// and a blob tx would contain target-num-frames many blobs.

	// It would be nicer to use core.IntrinsicGas, but we don't have the actual data at hand
	calldataBytes := dec.calldataConfig.MaxFrameSize + 1 // + 1 version byte
	calldataGas := big.NewInt(int64(calldataBytes*randomByteCalldataGas + params.TxGas))
	calldataPrice := new(big.Int).Add(baseFee, tipCap)
	calldataCost := new(big.Int).Mul(calldataGas, calldataPrice)

	blobGas := big.NewInt(params.BlobTxBlobGasPerBlob * int64(dec.blobConfig.TargetNumFrames))
	blobCost := new(big.Int).Mul(blobGas, blobBaseFee)
	// blobs still have intrinsic calldata costs
	blobCalldataCost := new(big.Int).Mul(big.NewInt(int64(params.TxGas)), calldataPrice)
	blobCost = blobCost.Add(blobCost, blobCalldataCost)

	// Now we compare the prices divided by the number of bytes that can be
	// submitted for that price.
	blobDataBytes := big.NewInt(eth.MaxBlobDataSize * int64(dec.blobConfig.TargetNumFrames))
	// The following will compare blobCost(a)/blobDataBytes(x) > calldataCost(b)/calldataBytes(y):
	ay := new(big.Int).Mul(blobCost, big.NewInt(int64(calldataBytes)))
	bx := new(big.Int).Mul(calldataCost, blobDataBytes)
	// ratio only used for logging, more correct multiplicative calculation used for comparison
	ayf, bxf := new(big.Float).SetInt(ay), new(big.Float).SetInt(bx)
	costRatio := new(big.Float).Quo(ayf, bxf)
	lgr := dec.log.New("base_fee", baseFee, "blob_base_fee", blobBaseFee, "tip_cap", tipCap,
		"calldata_bytes", calldataBytes, "calldata_cost", calldataCost,
		"blob_data_bytes", blobDataBytes, "blob_cost", blobCost,
		"cost_ratio", costRatio)

	if ay.Cmp(bx) > 0 {
		lgr.Info("Using calldata channel config")
		dec.lastConfig = &dec.calldataConfig
		return dec.calldataConfig
	}
	lgr.Info("Using blob channel config")
	dec.lastConfig = &dec.blobConfig
	return dec.blobConfig
}

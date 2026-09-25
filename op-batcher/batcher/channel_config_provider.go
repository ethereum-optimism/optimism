package batcher

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type (
	ChannelConfigProvider interface {
		ChannelConfig(isThrottling bool, isAmsterdam bool) ChannelConfig
	}

	GasPricer interface {
		SuggestGasPriceCaps(ctx context.Context) (tipCap *big.Int, baseFee *big.Int, blobTipCap *big.Int, blobBaseFee *big.Int, err error)
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

// ChannelConfig will perform an estimate of the cost per byte for
// calldata and for blobs, given current market conditions: it will return
// the appropriate ChannelConfig depending on which is cheaper. It makes
// assumptions about the typical makeup of channel data. isAmsterdam reports whether Amsterdam
// is active at the latest canonical L1 head.
//
// The blob config is returned when throttling is in progress, prioritizing throughput over cost
// in times of limited bandwidth.
func (dec *DynamicEthChannelConfig) ChannelConfig(isThrottling bool, isAmsterdam bool) ChannelConfig {
	if isThrottling {
		dec.log.Info("Using blob channel config while throttling is in progress")
		dec.lastConfig = &dec.blobConfig
		return dec.blobConfig
	}
	ctx, cancel := context.WithTimeout(context.Background(), dec.timeout)
	defer cancel()
	tipCap, baseFee, blobTipCap, blobBaseFee, err := dec.gasPricer.SuggestGasPriceCaps(ctx)
	if err != nil {
		dec.log.Warn("Error querying gas prices, returning last config", "err", err)
		return *dec.lastConfig
	}
	// Channels built for blobs have higher capacity than channels built for calldata.
	// If we have a channel built for calldata, we want to switch to blobs if the cost per byte is lower. Doing so
	// will mean a new channel is built which will not be full but will eventually fill up with additional data.
	// If we have a channel built for blobs, we similarly want to switch to calldata if the cost per byte is lower. Doing so
	// will mean several new (full) channels will be built resulting in several calldata txs. We compute the cost per byte
	// for a _single_ transaction in either case.

	// Before Amsterdam, we assume that compressed random channel data has few zeros and price every
	// byte as non-zero. Amsterdam charges the floor price equally for zero and non-zero bytes.
	calldataBytesPerTx := dec.calldataConfig.MaxFrameSize + 1 // +1 for the version byte
	numBlobsPerTx := dec.blobConfig.TargetNumFrames

	// Compute the total absolute cost of submitting either a single calldata tx or a single blob tx.
	calldataCost, blobCost, oracleBlobCost := computeSingleCalldataTxCost(uint64(calldataBytesPerTx), baseFee, tipCap, isAmsterdam),
		computeSingleBlobTxCost(numBlobsPerTx, baseFee, tipCap, blobBaseFee, isAmsterdam),
		computeSingleBlobTxCost(numBlobsPerTx, baseFee, blobTipCap, blobBaseFee, isAmsterdam)

	oracleBlobSavings := oracleBlobCost.Cmp(blobCost) < 0

	// Now we compare the absolute cost per tx divided by the number of bytes per tx:
	blobDataBytesPerTx := big.NewInt(eth.MaxBlobDataSize * int64(numBlobsPerTx))

	// The following will compare blobCost(a)/blobDataBytes(x) > calldataCost(b)/calldataBytes(y):
	ay := new(big.Int).Mul(blobCost, big.NewInt(int64(calldataBytesPerTx)))
	bx := new(big.Int).Mul(calldataCost, blobDataBytesPerTx)

	// ratio only used for logging, more correct multiplicative calculation used for comparison
	ayf, bxf := new(big.Float).SetInt(ay), new(big.Float).SetInt(bx)
	costRatio := new(big.Float).Quo(ayf, bxf)
	lgr := dec.log.New("base_fee", baseFee, "blob_base_fee", blobBaseFee, "tip_cap", tipCap,
		"amsterdam", isAmsterdam,
		"calldata_bytes", calldataBytesPerTx, "calldata_cost", calldataCost,
		"blob_data_bytes", blobDataBytesPerTx, "blob_cost", blobCost,
		"oracle_blob_cost", oracleBlobCost,
		"oracle_blob_savings", oracleBlobSavings,
		"cost_ratio", costRatio)

	if ay.Cmp(bx) == 1 {
		lgr.Info("Using calldata channel config")
		dec.lastConfig = &dec.calldataConfig
		return dec.calldataConfig
	}
	lgr.Info("Using blob channel config")
	dec.lastConfig = &dec.blobConfig
	return dec.blobConfig
}

const (
	pectraCalldataFloorGasPerByte    = params.TxTokenPerNonZeroByte * params.TxCostFloorPerToken // EIP-7623
	amsterdamTxBaseGas               = uint64(12_000 + 3_000)                                    // EIP-2780: TX_BASE_COST + COLD_ACCOUNT_ACCESS
	amsterdamCalldataFloorGasPerByte = uint64(4 * 16)                                            // EIP-7976: 4 floor tokens per byte at 16 gas per token
)

func computeSingleCalldataTxCost(numBytes uint64, baseFee, tipCap *big.Int, isAmsterdam bool) *big.Int {
	// Batch submissions send zero value to the code-less batch inbox, so their gas used is the
	// transaction base plus the calldata floor.
	txBaseGas := uint64(params.TxGas)
	calldataGasPerByte := pectraCalldataFloorGasPerByte
	if isAmsterdam {
		txBaseGas = amsterdamTxBaseGas
		calldataGasPerByte = amsterdamCalldataFloorGasPerByte
	}

	calldataPrice := new(big.Int).Add(baseFee, tipCap)
	calldataGas := new(big.Int).SetUint64(txBaseGas + numBytes*calldataGasPerByte)

	return new(big.Int).Mul(calldataGas, calldataPrice)
}

func computeSingleBlobTxCost(numBlobs int, baseFee, tipCap, blobBaseFee *big.Int, isAmsterdam bool) *big.Int {
	// Blob batch submissions have no calldata and send zero value to the code-less batch inbox.
	txBaseGas := uint64(params.TxGas)
	if isAmsterdam {
		txBaseGas = amsterdamTxBaseGas
	}
	calldataPrice := new(big.Int).Add(baseFee, tipCap)
	blobCalldataCost := new(big.Int).Mul(new(big.Int).SetUint64(txBaseGas), calldataPrice)

	blobGas := big.NewInt(params.BlobTxBlobGasPerBlob * int64(numBlobs))
	blobCost := new(big.Int).Mul(blobGas, blobBaseFee)

	return blobCost.Add(blobCost, blobCalldataCost)
}

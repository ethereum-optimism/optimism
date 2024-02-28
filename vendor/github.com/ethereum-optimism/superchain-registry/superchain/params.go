package superchain

import (
	"math/big"
)

var uint128Max, ok = big.NewInt(0).SetString("ffffffffffffffffffffffffffffffff", 16)

func init() {
	if !ok {
		panic("cannot construct uint128Max")
	}
}

type ResourceConfig struct {
	MaxResourceLimit            uint32
	ElasticityMultiplier        uint8
	BaseFeeMaxChangeDenominator uint8
	MinimumBaseFee              uint32
	SystemTxMaxGas              uint32
	MaximumBaseFee              *big.Int
}

// OPMainnetResourceConfig describes the resource metering configuration from OP Mainnet
var OPMainnetResourceConfig = ResourceConfig{
	MaxResourceLimit:            20000000,
	ElasticityMultiplier:        10,
	BaseFeeMaxChangeDenominator: 8,
	MinimumBaseFee:              1000000000,
	SystemTxMaxGas:              1000000,
	MaximumBaseFee:              uint128Max,
}

type L2OOParams struct {
	SubmissionInterval        *big.Int // Interval in blocks at which checkpoints must be submitted.
	L2BlockTime               *big.Int // The time per L2 block, in seconds.
	FinalizationPeriodSeconds *big.Int // The minimum time (in seconds) that must elapse before a withdrawal can be finalized.
}

// OPMainnetL2OOParams describes the L2OutputOracle parameters from OP Mainnet
var OPMainnetL2OOParams = L2OOParams{
	SubmissionInterval:        big.NewInt(120),
	L2BlockTime:               big.NewInt(2),
	FinalizationPeriodSeconds: big.NewInt(12),
}

// OPGoerliL2OOParams describes the L2OutputOracle parameters from OP Goerli
var OPGoerliL2OOParams = L2OOParams{
	SubmissionInterval:        big.NewInt(120),
	L2BlockTime:               big.NewInt(2),
	FinalizationPeriodSeconds: big.NewInt(12),
}

// OPGoerliDev0L2OOParams describes the L2OutputOracle parameters from OP Goerli
var OPGoerliDev0L2OOParams = L2OOParams{
	SubmissionInterval:        big.NewInt(120),
	L2BlockTime:               big.NewInt(2),
	FinalizationPeriodSeconds: big.NewInt(12),
}

// OPSepoliaL2OOParams describes the L2OutputOracle parameters from OP Goerli
var OPSepoliaL2OOParams = L2OOParams{
	SubmissionInterval:        big.NewInt(120),
	L2BlockTime:               big.NewInt(2),
	FinalizationPeriodSeconds: big.NewInt(12),
}

// OPSepoliaDev0L2OOParams describes the L2OutputOracle parameters from OP Goerli
var OPSepoliaDev0L2OOParams = L2OOParams{
	SubmissionInterval:        big.NewInt(120),
	L2BlockTime:               big.NewInt(2),
	FinalizationPeriodSeconds: big.NewInt(12),
}

type BigIntAndBounds struct {
	Value  *big.Int
	Bounds [2]*big.Int
}

type Uint32AndBounds struct {
	Value  uint32
	Bounds [2]uint32
}

type PreEcotoneGasPriceOracleParams struct {
	Decimals *big.Int
	Overhead *big.Int
	Scalar   *big.Int
}

type EcotoneGasPriceOracleParams struct {
	Decimals          *big.Int
	BlobBaseFeeScalar uint32
	BaseFeeScalar     uint32
}

type PreEcotoneGasPriceOracleParamsWithBounds struct {
	Decimals BigIntAndBounds
	Overhead BigIntAndBounds
	Scalar   BigIntAndBounds
}

type EcotoneGasPriceOracleParamsWithBounds struct {
	Decimals          BigIntAndBounds
	BlobBaseFeeScalar Uint32AndBounds
	BaseFeeScalar     Uint32AndBounds
}

type UpgradeFilter struct {
	PreEcotone *PreEcotoneGasPriceOracleParamsWithBounds
	Ecotone    *EcotoneGasPriceOracleParamsWithBounds
}

func makeBigIntAndBounds(value int64, bounds [2]int64) BigIntAndBounds {
	return BigIntAndBounds{big.NewInt(value), [2]*big.Int{big.NewInt(bounds[0]), big.NewInt(bounds[1])}}
}

var GasPriceOracleParams = map[string]UpgradeFilter{
	"mainnet": {
		PreEcotone: &PreEcotoneGasPriceOracleParamsWithBounds{
			Decimals: makeBigIntAndBounds(6, [2]int64{6, 6}),
			Overhead: makeBigIntAndBounds(188, [2]int64{188, 188}),
			Scalar:   makeBigIntAndBounds(684_000, [2]int64{684_000, 684_000}),
		},
	},
	"sepolia": {
		PreEcotone: &PreEcotoneGasPriceOracleParamsWithBounds{
			Decimals: makeBigIntAndBounds(6, [2]int64{6, 6}),
			Overhead: makeBigIntAndBounds(188, [2]int64{188, 2_100}),
			Scalar:   makeBigIntAndBounds(684_000, [2]int64{684_000, 1_000_000}),
		},
		Ecotone: &EcotoneGasPriceOracleParamsWithBounds{
			Decimals:          makeBigIntAndBounds(6, [2]int64{6, 6}),
			BlobBaseFeeScalar: Uint32AndBounds{862_000, [2]uint32{862_000, 862_000}},
			BaseFeeScalar:     Uint32AndBounds{7600, [2]uint32{7600, 7600}},
		},
	},
	"goerli": {
		PreEcotone: &PreEcotoneGasPriceOracleParamsWithBounds{
			Decimals: makeBigIntAndBounds(6, [2]int64{6, 6}),
			Overhead: makeBigIntAndBounds(2_100, [2]int64{2_100, 2_100}),
			Scalar:   makeBigIntAndBounds(100_000, [2]int64{100_000, 100_000}),
		},
		Ecotone: &EcotoneGasPriceOracleParamsWithBounds{
			Decimals:          makeBigIntAndBounds(6, [2]int64{6, 6}),
			BlobBaseFeeScalar: Uint32AndBounds{862_000, [2]uint32{862_000, 862_000}},
			BaseFeeScalar:     Uint32AndBounds{7600, [2]uint32{7600, 7600}},
		},
	},
}

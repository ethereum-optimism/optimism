package oracle

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/gas-oracle/flags"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
)

// Config represents the configuration options for the gas oracle
type Config struct {
	l1ChainID                    *big.Int
	l2ChainID                    *big.Int
	ethereumHttpUrl              string
	layerTwoHttpUrl              string
	gasPriceOracleAddress        common.Address
	privateKey                   *ecdsa.PrivateKey
	gasPrice                     *big.Int
	waitForReceipt               bool
	floorPrice                   uint64
	targetGasPerSecond           uint64
	maxPercentChangePerEpoch     float64
	averageBlockGasLimitPerEpoch uint64
	epochLengthSeconds           uint64
	l1BaseFeeEpochLengthSeconds  uint64
	l2GasPriceSignificanceFactor float64
	l1BaseFeeSignificanceFactor  float64
	enableL1BaseFee              bool
	enableL2GasPrice             bool
	// Metrics config
	MetricsEnabled          bool
	MetricsHTTP             string
	MetricsPort             int
	MetricsEnableInfluxDB   bool
	MetricsInfluxDBEndpoint string
	MetricsInfluxDBDatabase string
	MetricsInfluxDBUsername string
	MetricsInfluxDBPassword string
}

// NewConfig creates a new Config
func NewConfig(ctx *cli.Context) *Config {
	cfg := Config{}
	cfg.ethereumHttpUrl = ctx.GlobalString(flags.EthereumHttpUrlFlag.Name)
	cfg.layerTwoHttpUrl = ctx.GlobalString(flags.LayerTwoHttpUrlFlag.Name)
	addr := ctx.GlobalString(flags.GasPriceOracleAddressFlag.Name)
	cfg.gasPriceOracleAddress = common.HexToAddress(addr)
	cfg.targetGasPerSecond = ctx.GlobalUint64(flags.TargetGasPerSecondFlag.Name)
	cfg.maxPercentChangePerEpoch = ctx.GlobalFloat64(flags.MaxPercentChangePerEpochFlag.Name)
	cfg.averageBlockGasLimitPerEpoch = ctx.GlobalUint64(flags.AverageBlockGasLimitPerEpochFlag.Name)
	cfg.epochLengthSeconds = ctx.GlobalUint64(flags.EpochLengthSecondsFlag.Name)
	cfg.l1BaseFeeEpochLengthSeconds = ctx.GlobalUint64(flags.L1BaseFeeEpochLengthSecondsFlag.Name)
	cfg.l2GasPriceSignificanceFactor = ctx.GlobalFloat64(flags.L2GasPriceSignificanceFactorFlag.Name)
	cfg.floorPrice = ctx.GlobalUint64(flags.FloorPriceFlag.Name)
	cfg.l1BaseFeeSignificanceFactor = ctx.GlobalFloat64(flags.L1BaseFeeSignificanceFactorFlag.Name)
	cfg.enableL1BaseFee = ctx.GlobalBool(flags.EnableL1BaseFeeFlag.Name)
	cfg.enableL2GasPrice = ctx.GlobalBool(flags.EnableL2GasPriceFlag.Name)

	if ctx.GlobalIsSet(flags.PrivateKeyFlag.Name) {
		hex := ctx.GlobalString(flags.PrivateKeyFlag.Name)
		hex = strings.TrimPrefix(hex, "0x")
		key, err := crypto.HexToECDSA(hex)
		if err != nil {
			log.Error(fmt.Sprintf("Option %q: %v", flags.PrivateKeyFlag.Name, err))
		}
		cfg.privateKey = key
	} else {
		log.Crit("No private key configured")
	}

	if ctx.GlobalIsSet(flags.L1ChainIDFlag.Name) {
		chainID := ctx.GlobalUint64(flags.L1ChainIDFlag.Name)
		cfg.l1ChainID = new(big.Int).SetUint64(chainID)
	}
	if ctx.GlobalIsSet(flags.L2ChainIDFlag.Name) {
		chainID := ctx.GlobalUint64(flags.L2ChainIDFlag.Name)
		cfg.l2ChainID = new(big.Int).SetUint64(chainID)
	}

	if ctx.GlobalIsSet(flags.TransactionGasPriceFlag.Name) {
		gasPrice := ctx.GlobalUint64(flags.TransactionGasPriceFlag.Name)
		cfg.gasPrice = new(big.Int).SetUint64(gasPrice)
	}

	if ctx.GlobalIsSet(flags.WaitForReceiptFlag.Name) {
		cfg.waitForReceipt = true
	}

	cfg.MetricsEnabled = ctx.GlobalBool(flags.MetricsEnabledFlag.Name)
	cfg.MetricsHTTP = ctx.GlobalString(flags.MetricsHTTPFlag.Name)
	cfg.MetricsPort = ctx.GlobalInt(flags.MetricsPortFlag.Name)
	cfg.MetricsEnableInfluxDB = ctx.GlobalBool(flags.MetricsEnableInfluxDBFlag.Name)
	cfg.MetricsInfluxDBEndpoint = ctx.GlobalString(flags.MetricsInfluxDBEndpointFlag.Name)
	cfg.MetricsInfluxDBDatabase = ctx.GlobalString(flags.MetricsInfluxDBDatabaseFlag.Name)
	cfg.MetricsInfluxDBUsername = ctx.GlobalString(flags.MetricsInfluxDBUsernameFlag.Name)
	cfg.MetricsInfluxDBPassword = ctx.GlobalString(flags.MetricsInfluxDBPasswordFlag.Name)

	return &cfg
}

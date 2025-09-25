package derive

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// Gas Price Oracle Parameters
	deployJovianGasPriceOracleSource    = UpgradeDepositSource{Intent: "Jovian: Gas Price Oracle Deployment"}
	updateJovianGasPriceOracleSource    = UpgradeDepositSource{Intent: "Jovian: Gas Price Oracle Proxy Update"}
	GasPriceOracleJovianDeployerAddress = common.HexToAddress("0x4210000000000000000000000000000000000006")
	jovianGasPriceOracleAddress         = crypto.CreateAddress(GasPriceOracleJovianDeployerAddress, 0)

	// Bytecodes
	gasPriceOracleJovianDeploymentBytecode = common.FromHex("0xTODO")

	// Enable Jovian Parameters
	enableJovianSource = UpgradeDepositSource{Intent: "Jovian: Gas Price Oracle Set Jovian"}
	enableJovianInput  = crypto.Keccak256([]byte("setJovian()"))[:4]
)

func JovianNetworkUpgradeTransactions() ([]hexutil.Bytes, error) {
	upgradeTxns := make([]hexutil.Bytes, 0, 8)

	// Deploy Gas Price Oracle transaction
	deployGasPriceOracle, err := types.NewTx(&types.DepositTx{
		SourceHash:          deployJovianGasPriceOracleSource.SourceHash(),
		From:                GasPriceOracleJovianDeployerAddress,
		To:                  nil,
		Mint:                big.NewInt(0),
		Value:               big.NewInt(0),
		Gas:                 1_625_000,
		IsSystemTransaction: false,
		Data:                gasPriceOracleJovianDeploymentBytecode,
	}).MarshalBinary()

	if err != nil {
		return nil, err
	}

	upgradeTxns = append(upgradeTxns, deployGasPriceOracle)

	// Deploy Gas Price Oracle Proxy upgrade transaction
	updateGasPriceOracleProxy, err := types.NewTx(&types.DepositTx{
		SourceHash:          updateJovianGasPriceOracleSource.SourceHash(),
		From:                common.Address{},
		To:                  &predeploys.GasPriceOracleAddr,
		Mint:                big.NewInt(0),
		Value:               big.NewInt(0),
		Gas:                 50_000,
		IsSystemTransaction: false,
		Data:                upgradeToCalldata(jovianGasPriceOracleAddress),
	}).MarshalBinary()

	if err != nil {
		return nil, err
	}

	upgradeTxns = append(upgradeTxns, updateGasPriceOracleProxy)

	// Enable Jovian transaction
	enableJovian, err := types.NewTx(&types.DepositTx{
		SourceHash:          enableJovianSource.SourceHash(),
		From:                L1InfoDepositerAddress,
		To:                  &predeploys.GasPriceOracleAddr,
		Mint:                big.NewInt(0),
		Value:               big.NewInt(0),
		Gas:                 90_000,
		IsSystemTransaction: false,
		Data:                enableJovianInput,
	}).MarshalBinary()

	if err != nil {
		return nil, err
	}

	upgradeTxns = append(upgradeTxns, enableJovian)

	return upgradeTxns, nil
}

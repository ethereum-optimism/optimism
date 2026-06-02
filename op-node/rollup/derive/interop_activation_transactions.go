package derive

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Interop activates via a JSON NUT bundle (see UpgradeTransactions(forks.Interop))
// that may be wrapped between two hardcoded deposit transactions:
//
//	[1] interopSetFeatureTx           — must run before the bundle so the L2CM
//	                                    upgrade reads isFeatureEnabled(INTEROP)=true
//	[N] interopETHLiquidityFundingTx  — runs after the bundle; the only Interop
//	                                    deposit with non-zero mint and value
//	                                    (max uint128), so it cannot be expressed
//	                                    in the JSON schema.
const (
	interopSetFeatureGas       uint64 = 100_000
	interopETHLiquidityFundGas uint64 = 50_000
)

var (
	interopSetFeatureSource     = UpgradeDepositSource{Intent: "Interop pre: setFeature(INTEROP)"}
	interopETHLiquidityFundSrc  = UpgradeDepositSource{Intent: "Interop post: ETHLiquidity Funding"}
	interopETHLiquidityFundData = crypto.Keccak256([]byte("fund()"))[:4]
)

// InteropActivationUpgradeTransactions returns the Interop activation deposits.
// The NUT bundle always executes. The setFeature and ETHLiquidity funding
// wrappers execute only when activateInteropContracts is true.
func InteropActivationUpgradeTransactions(activateInteropContracts bool) ([]hexutil.Bytes, uint64, error) {
	bundleTxs, bundleGas, err := UpgradeTransactions(forks.Interop)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load interop NUT bundle: %w", err)
	}

	if !activateInteropContracts {
		return bundleTxs, bundleGas, nil
	}

	setFeatureTx, err := interopSetFeatureTx()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build interop setFeature wrapper: %w", err)
	}
	fundingTx, err := interopETHLiquidityFundingTx()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build interop ETHLiquidity funding wrapper: %w", err)
	}

	txs := make([]hexutil.Bytes, 0, 2+len(bundleTxs))
	txs = append(txs, setFeatureTx)
	txs = append(txs, bundleTxs...)
	txs = append(txs, fundingTx)
	return txs, interopSetFeatureGas + bundleGas + interopETHLiquidityFundGas, nil
}

// InteropETHLiquidityFundingAmount returns the bootstrap liquidity minted into the
// ETHLiquidity contract at Interop activation: the maximum uint128 value, which is
// also the maximum the deposit-tx mint field supports.
func InteropETHLiquidityFundingAmount() *big.Int {
	v, _ := new(big.Int).SetString("ffffffffffffffffffffffffffffffff", 16)
	return v
}

// interopSetFeatureTx returns the encoded pre-bundle setFeature(INTEROP) deposit.
// It flips L1Block.isFeatureEnabled(INTEROP) so that the L2CM upgrade
// (executed inside the bundle's last tx) applies the Interop-gated proxy upgrades.
func interopSetFeatureTx() (hexutil.Bytes, error) {
	selector := crypto.Keccak256([]byte("setFeature(bytes32)"))[:4]
	var featureBytes [32]byte
	copy(featureBytes[:], "INTEROP")
	data := make([]byte, 0, len(selector)+32)
	data = append(data, selector...)
	data = append(data, featureBytes[:]...)

	addr := predeploys.L1BlockAddr
	return types.NewTx(&types.DepositTx{
		SourceHash:          interopSetFeatureSource.SourceHash(),
		From:                L1InfoDepositerAddress,
		To:                  &addr,
		Mint:                big.NewInt(0),
		Value:               big.NewInt(0),
		Gas:                 interopSetFeatureGas,
		IsSystemTransaction: false,
		Data:                data,
	}).MarshalBinary()
}

// interopETHLiquidityFundingTx returns the encoded post-bundle ETHLiquidity funding
// deposit. The mint and value are u128::MAX — the only Interop deposit with
// non-zero mint/value, hence not expressible in the JSON NUT bundle schema.
func interopETHLiquidityFundingTx() (hexutil.Bytes, error) {
	addr := predeploys.ETHLiquidityAddr
	amount := InteropETHLiquidityFundingAmount()
	return types.NewTx(&types.DepositTx{
		SourceHash:          interopETHLiquidityFundSrc.SourceHash(),
		From:                L1InfoDepositerAddress,
		To:                  &addr,
		Mint:                amount,
		Value:               amount,
		Gas:                 interopETHLiquidityFundGas,
		IsSystemTransaction: false,
		Data:                interopETHLiquidityFundData,
	}).MarshalBinary()
}

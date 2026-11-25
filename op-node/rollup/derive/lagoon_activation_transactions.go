package derive

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// Interop activates via a JSON NUT bundle (see UpgradeTransactions(forks.Lagoon))
// that may be wrapped between two hardcoded deposit transactions:
//
//	[1] interopSetFeatureDeposit           — must run before the bundle so the L2CM
//	                                    upgrade reads isFeatureEnabled(INTEROP)=true
//	[N] interopETHLiquidityFundingDeposit  — runs after the bundle; the only Interop
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

// LagoonActivationUpgradeTransactions returns the Lagoon activation deposits and the gas to add to
// the activation block's gas limit. The NUT bundle always executes. The setFeature and ETHLiquidity
// funding wrappers execute only when activateInteropContracts is true.
//
// The gas covers the wrappers either way, so that the amount is independent of the dependency set.
// See UpgradeGas.
func LagoonActivationUpgradeTransactions(activateInteropContracts bool) ([]hexutil.Bytes, uint64, error) {
	bundleTxs, gas, err := UpgradeTransactions(forks.Lagoon)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load Lagoon NUT bundle: %w", err)
	}

	if !activateInteropContracts {
		return bundleTxs, gas, nil
	}

	setFeatureTx, err := interopSetFeatureDeposit().MarshalBinary()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build interop setFeature wrapper: %w", err)
	}
	fundingTx, err := interopETHLiquidityFundingDeposit().MarshalBinary()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build interop ETHLiquidity funding wrapper: %w", err)
	}

	txs := make([]hexutil.Bytes, 0, 2+len(bundleTxs))
	txs = append(txs, setFeatureTx)
	txs = append(txs, bundleTxs...)
	txs = append(txs, fundingTx)
	return txs, gas, nil
}

// interopWrapperGas is the gas the wrapper deposits occupy in the activation block. It is summed
// from the deposits themselves, so the reservation cannot drift from what they actually cost.
func interopWrapperGas() uint64 {
	return interopSetFeatureDeposit().Gas + interopETHLiquidityFundingDeposit().Gas
}

// InteropETHLiquidityFundingAmount returns the bootstrap liquidity minted into the
// ETHLiquidity contract at Interop activation: the maximum uint128 value, which is
// also the maximum the deposit-tx mint field supports.
func InteropETHLiquidityFundingAmount() *big.Int {
	v, _ := new(big.Int).SetString("ffffffffffffffffffffffffffffffff", 16)
	return v
}

// interopSetFeatureDeposit returns the pre-bundle setFeature(INTEROP) deposit.
// It flips L1Block.isFeatureEnabled(INTEROP) so that the L2CM upgrade
// (executed inside the bundle's last tx) applies the Interop-gated proxy upgrades.
func interopSetFeatureDeposit() *optypes.DepositTx {
	selector := crypto.Keccak256([]byte("setFeature(bytes32)"))[:4]
	var featureBytes [32]byte
	copy(featureBytes[:], "INTEROP")
	data := make([]byte, 0, len(selector)+32)
	data = append(data, selector...)
	data = append(data, featureBytes[:]...)

	addr := predeploys.L1BlockAddr
	return &optypes.DepositTx{
		SourceHash:          interopSetFeatureSource.SourceHash(),
		From:                L1InfoDepositerAddress,
		To:                  &addr,
		Mint:                big.NewInt(0),
		Value:               big.NewInt(0),
		Gas:                 interopSetFeatureGas,
		IsSystemTransaction: false,
		Data:                data,
	}
}

// interopETHLiquidityFundingDeposit returns the post-bundle ETHLiquidity funding
// deposit. The mint and value are u128::MAX — the only Lagoon activation deposit
// with a non-zero mint, hence not expressible in the JSON NUT bundle schema.
func interopETHLiquidityFundingDeposit() *optypes.DepositTx {
	addr := predeploys.ETHLiquidityAddr
	amount := InteropETHLiquidityFundingAmount()
	return &optypes.DepositTx{
		SourceHash:          interopETHLiquidityFundSrc.SourceHash(),
		From:                L1InfoDepositerAddress,
		To:                  &addr,
		Mint:                amount,
		Value:               amount,
		Gas:                 interopETHLiquidityFundGas,
		IsSystemTransaction: false,
		Data:                interopETHLiquidityFundData,
	}
}

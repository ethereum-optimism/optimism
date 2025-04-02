package operatorfee

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/isthmus"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var returnRemainingFundsGasFeeCap = big.NewInt(500_000_000_000)
var returnRemainingFundsGasTipCap = big.NewInt(1)

func SendValueTx(wallet system.WalletV2, to common.Address, value *big.Int) (tx *gethTypes.Transaction, receipt *gethTypes.Receipt, err error) {
	opts := isthmus.DefaultTxOpts(wallet)
	deployTx := txplan.NewPlannedTx(opts,
		txplan.WithValue(value),
		txplan.WithTo(&to),
	)
	signedTx, err := deployTx.Signed.Eval(wallet.Ctx())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign tx: %w", err)
	}
	_, err = deployTx.Submitted.Eval(wallet.Ctx())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to submit tx: %w", err)
	}

	receipt, err = deployTx.Included.Eval(wallet.Ctx())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find receipt for tx %s: %w", signedTx.Hash().Hex(), err)
	}
	return signedTx, receipt, nil
}

func ReturnRemainingFunds(wallet system.WalletV2, to common.Address) (receipt *gethTypes.Receipt, err error) {
	balance, err := wallet.Client().BalanceAt(wallet.Ctx(), wallet.Address(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}
	value := balance.Sub(balance, new(big.Int).Mul(returnRemainingFundsGasFeeCap, big.NewInt(21000)))

	if value.Sign() <= 0 {
		return nil, nil
	}

	deployTx := txplan.NewPlannedTx(
		txplan.WithRetryInclusion(wallet.Client(), 30, retry.Exponential()),
		txplan.WithBlockInclusionInfo(wallet.Client()),
		txplan.WithPrivateKey(wallet.PrivateKey()),
		txplan.WithChainID(wallet.Client()),
		txplan.WithAgainstLatestBlock(wallet.Client()),
		txplan.WithPendingNonce(wallet.Client()),
		txplan.WithGasLimit(21000),
		txplan.WithGasFeeCap(returnRemainingFundsGasFeeCap),
		txplan.WithGasTipCap(returnRemainingFundsGasTipCap),
		txplan.WithTransactionSubmitter(wallet.Client()),
		txplan.WithValue(value),
		txplan.WithTo(&to),
	)
	return deployTx.Included.Eval(wallet.Ctx())
}

func NewTestWallet(ctx context.Context, chain system.Chain) (system.Wallet, error) {
	// create new test wallet
	testWalletPrivateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	testWalletPrivateKeyBytes := crypto.FromECDSA(testWalletPrivateKey)
	testWalletPrivateKeyHex := hex.EncodeToString(testWalletPrivateKeyBytes)
	testWalletPublicKey := testWalletPrivateKey.Public()
	testWalletPublicKeyECDSA, ok := testWalletPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Failed to assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	testWalletAddress := crypto.PubkeyToAddress(*testWalletPublicKeyECDSA)
	testWallet, err := system.NewWallet(
		testWalletPrivateKeyHex,
		types.Address(testWalletAddress),
		chain,
	)
	return testWallet, err
}

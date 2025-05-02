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
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func EnsureSufficientBalance(wallet system.WalletV2, to common.Address, value *big.Int) (err error) {
	balance, err := wallet.Client().BalanceAt(wallet.Ctx(), to, nil)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}
	if balance.Cmp(value) < 0 {
		tx, receipt, err := SendValueTx(wallet, to, value)
		if err != nil {
			return fmt.Errorf("failed to send value tx: %w", err)
		}
		if receipt.Status != gethTypes.ReceiptStatusSuccessful {
			return fmt.Errorf("tx %s failed with status %d", tx.Hash().Hex(), receipt.Status)
		}
	}
	return nil
}

func SendValueTx(wallet system.WalletV2, to common.Address, value *big.Int) (tx *gethTypes.Transaction, receipt *gethTypes.Receipt, err error) {
	// ensure wallet is not the same as to address
	if wallet.Address() == to {
		return nil, nil, fmt.Errorf("wallet address is the same as the to address")
	}

	walletPreBalance, err := wallet.Client().BalanceAt(wallet.Ctx(), wallet.Address(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get balance for from address: %w", err)
	}
	receiverPreBalance, err := wallet.Client().BalanceAt(wallet.Ctx(), to, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get balance for to address: %w", err)
	}
	if walletPreBalance.Cmp(value) < 0 {
		return nil, nil, fmt.Errorf("sender (%s) balance (%s) is less than the value (%s) attempting to send", wallet.Address(), walletPreBalance.String(), value.String())
	}

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

	_, err = deployTx.Success.Eval(wallet.Ctx())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check tx success: %w", err)
	}

	receipt = deployTx.Included.Value()

	// verify balance of wallet
	blockNumber := new(big.Int).SetUint64(receipt.BlockNumber.Uint64())
	receiverPostBalance, err := wallet.Client().BalanceAt(wallet.Ctx(), to, blockNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get to post-balance: %w", err)
	}
	if new(big.Int).Sub(receiverPostBalance, receiverPreBalance).Cmp(value) != 0 {
		return nil, nil, fmt.Errorf("wallet balance was not updated successfully, expected %s, got %s", new(big.Int).Add(receiverPreBalance, value).String(), receiverPostBalance.String())
	}

	return signedTx, receipt, nil
}

func ReturnRemainingFunds(wallet system.WalletV2, to common.Address) (receipt *gethTypes.Receipt, err error) {
	balance, err := wallet.Client().BalanceAt(wallet.Ctx(), wallet.Address(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	opts := isthmus.DefaultTxOpts(wallet)
	txPlan := txplan.NewPlannedTx(opts,
		txplan.WithTo(&to),
	)
	innerTx, err := txPlan.Unsigned.Eval(wallet.Ctx())
	if err != nil {
		return nil, fmt.Errorf("failed to get inner tx: %w", err)
	}

	dynInnerTx, ok := innerTx.(*gethTypes.DynamicFeeTx)
	if !ok {
		return nil, fmt.Errorf("inner tx is not a dynamic fee tx")
	}

	gasLimit := dynInnerTx.Gas
	gasFeeCap := dynInnerTx.GasFeeCap
	gasCost := new(big.Int).Mul(big.NewInt(int64(gasLimit)), gasFeeCap)

	value := new(big.Int).Sub(balance, gasCost)

	if value.Sign() < 0 {
		// insufficient balance, so we don't need to send a tx
		return nil, nil
	}

	dynInnerTx.Value = value

	opts = isthmus.DefaultTxOpts(wallet)
	txPlan = txplan.NewPlannedTx(opts,
		txplan.WithUnsigned(dynInnerTx),
	)

	_, err = txPlan.Success.Eval(wallet.Ctx())
	if err != nil {
		return nil, fmt.Errorf("return remaining funds tx %s failed: %w", txPlan.Signed.Value().Hash().Hex(), err)
	}

	receipt = txPlan.Included.Value()

	return receipt, nil
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

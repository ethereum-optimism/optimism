package interop

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	sdktypes "github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/stretchr/testify/require"
)

func DefaultTxSubmitOptions(w system.WalletV2) txplan.Option {
	return txplan.Combine(
		txplan.WithPrivateKey(w.PrivateKey()),
		txplan.WithChainID(w.Client()),
		txplan.WithAgainstLatestBlock(w.Client()),
		txplan.WithPendingNonce(w.Client()),
		txplan.WithEstimator(w.Client(), false),
		txplan.WithTransactionSubmitter(w.Client()),
	)
}

func DefaultTxInclusionOptions(w system.WalletV2) txplan.Option {
	return txplan.Combine(
		txplan.WithRetryInclusion(w.Client(), 10, retry.Exponential()),
		txplan.WithBlockInclusionInfo(w.Client()),
	)
}

func DefaultTxOpts(w system.WalletV2) txplan.Option {
	return txplan.Combine(
		DefaultTxSubmitOptions(w),
		DefaultTxInclusionOptions(w),
	)
}

func GetWalletV2AndOpts(ctx context.Context, walletGetter validators.WalletGetter, chain system.Chain) (system.WalletV2, txplan.Option, error) {
	wallet, err := system.NewWalletV2FromWalletAndChain(ctx, walletGetter(ctx), chain)
	if err != nil {
		return nil, nil, err
	}
	opts := DefaultTxOpts(wallet)
	return wallet, opts, nil
}

func DefaultInteropSetup(t systest.T,
	sys system.InteropSystem,
	l2ChainNums int,
	walletGetters []validators.WalletGetter,
) (context.Context, *rand.Rand, log.Logger, []system.Chain, []system.WalletV2, []txplan.Option) {
	ctx := t.Context()
	rng := rand.New(rand.NewSource(1234))
	logger := testlog.Logger(t, log.LevelInfo)

	chains := make([]system.Chain, l2ChainNums)
	wallets := make([]system.WalletV2, 0)
	opts := make([]txplan.Option, 0)
	for idx := range l2ChainNums {
		chain := sys.L2s()[idx]
		chains = append(chains, chain)
		wallet, opt, err := GetWalletV2AndOpts(ctx, walletGetters[idx], chain)
		require.NoError(t, err)
		wallets = append(wallets, wallet)
		opts = append(opts, opt)
	}
	return ctx, rng, logger, chains, wallets, opts
}

func DeployEventLogger(ctx context.Context, wallet system.WalletV2, logger log.Logger) (common.Address, error) {
	opts := DefaultTxOpts(wallet)
	logger.Info("Deploying EventLogger")
	deployCalldata := common.FromHex(bindings.EventloggerBin)
	deployTx := txplan.NewPlannedTx(opts, txplan.WithData(deployCalldata))

	res, err := deployTx.Included.Eval(ctx)
	if err != nil {
		return common.Address{}, err
	}
	eventLoggerAddress := res.ContractAddress
	logger.Info("Deployed EventLogger", "chainID", deployTx.ChainID.Value(), "address", eventLoggerAddress)
	return eventLoggerAddress, err
}

func RandomTopicAndData(rng *rand.Rand, cnt, len int) ([][32]byte, []byte) {
	topics := [][32]byte{}
	for range cnt {
		var topic [32]byte
		copy(topic[:], testutils.RandomData(rng, 32))
		topics = append(topics, topic)
	}
	data := testutils.RandomData(rng, len)
	return topics, data
}

func RandomInitTrigger(rng *rand.Rand, eventLoggerAddress common.Address, cnt, len int) *txintent.InitTrigger {
	topics, data := RandomTopicAndData(rng, cnt, len)
	return &txintent.InitTrigger{
		Emitter:    eventLoggerAddress,
		Topics:     topics,
		OpaqueData: data,
	}
}

func SetupDefaultInteropSystemTest(l2ChainNums int) ([]validators.WalletGetter, []systest.PreconditionValidator) {
	walletGetters := make([]validators.WalletGetter, l2ChainNums)
	totalValidators := make([]systest.PreconditionValidator, 0)
	for i := range l2ChainNums {
		walletGetter, fundsValidator := validators.AcquireL2WalletWithFunds(
			uint64(i), sdktypes.NewBalance(big.NewInt(1*constants.ETH)),
		)
		walletGetters[i] = walletGetter
		totalValidators = append(totalValidators, fundsValidator)
	}
	return walletGetters, totalValidators
}

// FundWalletsFromFaucet funds wallet and returns txplan options which encapsulates funded wallet
func FundWalletFromFaucet(ctx context.Context, logger log.Logger, sys system.InteropSystem, chainIdx int, faucetOpt txplan.Option, amount *big.Int) (txplan.Option, error) {
	if len(sys.L2s()) < chainIdx {
		return nil, fmt.Errorf("invalid chain idx: %d", chainIdx)
	}
	chain := sys.L2s()[chainIdx]
	if len(chain.Nodes()) == 0 {
		return nil, fmt.Errorf("no node available at chain: %s", chain.ID())
	}
	node := chain.Nodes()[0]
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	wallet, err := system.NewWalletV2(ctx, node.RPCURL(), privateKey, nil, logger)
	if err != nil {
		return nil, err
	}
	to := crypto.PubkeyToAddress(privateKey.PublicKey)
	opt := DefaultTxOpts(wallet)
	tx := txplan.NewPlannedTx(faucetOpt, txplan.WithValue(amount), txplan.WithTo(&to))
	_, err = tx.Included.Eval(ctx)
	if err != nil {
		return nil, err
	}
	logger.Info("Funded", "address", to, "chainID", chain.ID())
	return opt, nil
}

// FundWalletsFromFaucet funds wallets and returns txplan options which encapsulates funded wallets
func FundWalletsFromFaucet(ctx context.Context, logger log.Logger, sys system.InteropSystem, chainIdx int, faucetOpt txplan.Option, amount *big.Int, cnt int) ([]txplan.Option, error) {
	opts := []txplan.Option{}
	for range cnt {
		opt, err := FundWalletFromFaucet(ctx, logger, sys, chainIdx, faucetOpt, amount)
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	}
	return opts, nil
}

// InitiateRandomMessages batches random messages and initiates them via a single multicall
func InitiateRandomMessages(ctx context.Context, opt txplan.Option, rng *rand.Rand, eventLoggerAddress common.Address) (*txintent.IntentTx[*txintent.MultiTrigger, *txintent.InteropOutput], *types.Receipt, error) {
	// Intent to initiate messages
	eventCnt := 1 + rng.Intn(9)
	initCalls := make([]txintent.Call, eventCnt)
	for index := range eventCnt {
		initCalls[index] = RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(5), rng.Intn(100))
	}

	tx := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](opt)
	tx.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: initCalls})

	// Trigger multiple events
	receipt, err := tx.PlannedTx.Included.Eval(ctx)
	if err != nil {
		return nil, nil, err
	}
	return tx, receipt, nil
}

// ValidateEveryMessage batches every message and validates them via a single multicall
func ValidateEveryMessage(ctx context.Context, opt txplan.Option, dependOn *txintent.IntentTx[*txintent.MultiTrigger, *txintent.InteropOutput]) (*txintent.IntentTx[*txintent.MultiTrigger, *txintent.InteropOutput], *types.Receipt, error) {
	// Intent to validate message
	tx := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](opt)
	tx.Content.DependOn(&dependOn.Result)

	indexes := []int{}
	result, err := dependOn.Result.Eval(ctx)
	if err != nil {
		return nil, nil, err
	}
	for idx := range len(result.Entries) {
		indexes = append(indexes, idx)
	}
	tx.Content.Fn(txintent.ExecuteIndexeds(constants.MultiCall3, constants.CrossL2Inbox, &dependOn.Result, indexes))
	receipt, err := tx.PlannedTx.Included.Eval(ctx)
	if err != nil {
		return nil, nil, err
	}
	return tx, receipt, err
}

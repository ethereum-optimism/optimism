package interop

import (
	"context"
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
	logger.Info("Deployed EventLogger", "address", eventLoggerAddress)
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

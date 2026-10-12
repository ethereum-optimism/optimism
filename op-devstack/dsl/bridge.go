package dsl

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	nodebindings "github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// ProvenWithdrawalParameters is the set of parameters to pass to the ProveWithdrawalTransaction
// and FinalizeWithdrawalTransaction functions
type ProvenWithdrawalParameters struct {
	Nonce              *big.Int
	Sender             common.Address
	Target             common.Address
	Value              *big.Int
	GasLimit           *big.Int
	DisputeGameAddress common.Address
	DisputeGameIndex   *big.Int
	Data               []byte
	OutputRootProof    bindings.OutputRootProof
	WithdrawalProof    [][]byte // List of trie nodes to prove L2 storage
}

type StandardBridge struct {
	commonImpl
	l1PortalAddr        common.Address
	l1Portal            bindings.OptimismPortal2
	l2tol1MessagePasser bindings.L2ToL1MessagePasser
	disputeGameFactory  bindings.DisputeGameFactory
	rollupCfg           *rollup.Config

	l1Client *L1ELNode
	l2Client apis.EthClient
	l2EL     *L2ELNode

	// L1 bridge contract
	l1StandardBridge bindings.L1StandardBridge
}

func NewStandardBridge(t devtest.T, l2Network *L2Network, l1EL *L1ELNode) *StandardBridge {
	l1Client := l1EL.EthClient()
	l1PortalAddr := l2Network.DepositContractAddr()
	l1Portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(l1Client),
		bindings.WithTo(l1PortalAddr),
		bindings.WithTest(t))
	l2Client := l2Network.PrimaryEL().EthClient()
	l2tol1MessagePasser := bindings.NewBindings[bindings.L2ToL1MessagePasser](
		bindings.WithClient(l2Client),
		bindings.WithTo(predeploys.L2ToL1MessagePasserAddr),
		bindings.WithTest(t))

	disputeGameFactory := bindings.NewBindings[bindings.DisputeGameFactory](
		bindings.WithClient(l1Client),
		bindings.WithTo(l2Network.DisputeGameFactoryProxyAddr()))

	l1StandardBridge := bindings.NewBindings[bindings.L1StandardBridge](
		bindings.WithClient(l1Client),
		bindings.WithTo(l2Network.Escape().Deployment().L1StandardBridgeProxyAddr()),
		bindings.WithTest(t))

	return &StandardBridge{
		commonImpl:          commonFromT(t),
		l1PortalAddr:        l1PortalAddr,
		l1Portal:            l1Portal,
		l2tol1MessagePasser: l2tol1MessagePasser,
		disputeGameFactory:  disputeGameFactory,
		rollupCfg:           l2Network.inner.RollupConfig(),

		l1Client:         l1EL,
		l2Client:         l2Client,
		l2EL:             l2Network.PrimaryEL(),
		l1StandardBridge: l1StandardBridge,
	}
}

func (b *StandardBridge) GameResolutionDelay() time.Duration {
	gameType := b.RespectedGameType()
	if gameTypes.GameType(gameType) == gameTypes.SuperPermissionedGameType {
		return 0
	}
	gameImplAddr, err := contractio.Read(b.disputeGameFactory.GameImpls(gameType), b.ctx)
	b.require.NoErrorf(err, "failed to get implementation for game type %v", gameType)
	game := bindings.NewBindings[bindings.FaultDisputeGame](bindings.WithClient(b.l1Client.EthClient()), bindings.WithTo(gameImplAddr), bindings.WithTest(b.t))
	clockDuration, err := contractio.Read(game.MaxClockDuration(), b.ctx)
	b.require.NoErrorf(err, "failed to get max clock duration for game type %v", gameType)
	return time.Duration(clockDuration) * time.Second
}

func (b *StandardBridge) WithdrawalDelay() time.Duration {
	delaySeconds, err := contractio.Read(b.l1Portal.ProofMaturityDelaySeconds(), b.ctx)
	b.require.NoError(err, "Failed to read proof maturity delay")
	return time.Duration(delaySeconds.Int64()) * time.Second
}

func (b *StandardBridge) DisputeGameFinalityDelay() time.Duration {
	delaySeconds, err := contractio.Read(b.l1Portal.DisputeGameFinalityDelaySeconds(), b.ctx)
	b.require.NoError(err, "Failed to read dispute game finality delay")
	return time.Duration(delaySeconds.Int64()) * time.Second
}

func (b *StandardBridge) RespectedGameType() uint32 {
	gameType, err := contractio.Read(b.l1Portal.RespectedGameType(), b.ctx)
	b.require.NoError(err, "Failed to read respected game type")
	return gameType
}

func (b *StandardBridge) VerifyRespectedGameType(expected gameTypes.GameType) {
	actual := gameTypes.GameType(b.RespectedGameType())
	b.require.Equalf(expected, actual,
		"respected game type mismatch: expected %s (%d), got %s (%d)",
		expected, uint32(expected), actual, uint32(actual))
}

func (b *StandardBridge) PortalVersion() string {
	version, err := contractio.Read(b.l1Portal.Version(), b.ctx)
	b.require.NoError(err, "Failed to read portal version")
	return version
}

func (b *StandardBridge) UsesSuperRoots() bool {
	gameType := gameTypes.GameType(b.RespectedGameType())
	return gameType == gameTypes.SuperPermissionedGameType ||
		gameType == gameTypes.SuperAsteriscKonaGameType ||
		gameType == gameTypes.SuperCannonKonaGameType ||
		gameType == gameTypes.ZKDisputeGameType
}

type Deposit struct {
	bridge    *StandardBridge
	l1Receipt *types.Receipt
}

func (d Deposit) GasCost() eth.ETH {
	if d.bridge == nil {
		panic("bridge reference not set on deposit")
	}
	return d.bridge.gasCost(d.l1Receipt, d.bridge.l1Client.EthClient())
}

func (b *StandardBridge) Deposit(amount eth.ETH, from *EOA) Deposit {
	depositTx := from.Transfer(b.l1PortalAddr, amount)
	l1DepositReceipt, err := depositTx.Included.Eval(b.ctx)
	b.require.NoErrorf(err, "Failed to send deposit transaction from %v for %v", from, amount)

	// Wait for the deposit to be processed on the L2
	// Construct the L2 deposit tx to check the tx is included at L2
	idx := len(l1DepositReceipt.Logs) - 1
	l2DepositTx, err := derive.UnmarshalDepositLogEvent(l1DepositReceipt.Logs[idx])
	b.require.NoError(err, "Could not reconstruct L2 Deposit")
	l2DepositTxHash := l2DepositTx.Hash()
	// Give time for L2CL to include the L2 deposit tx
	var l2DepositReceipt *optypes.Receipt
	b.require.Eventually(func() bool {
		l2DepositReceipt, err = b.l2Client.TransactionReceipt(b.ctx, l2DepositTxHash)
		return err == nil
	}, 60*time.Second, 500*time.Millisecond, "L2 Deposit never found")
	b.require.Equal(types.ReceiptStatusSuccessful, l2DepositReceipt.Status)
	return Deposit{
		bridge:    b,
		l1Receipt: l1DepositReceipt,
	}
}

func (b *StandardBridge) InitiateWithdrawal(amount eth.ETH, from *EOA) *Withdrawal {
	withdrawTx := from.Transfer(predeploys.L2ToL1MessagePasserAddr, amount)
	withdrawRcpt, err := withdrawTx.Included.Eval(b.ctx)
	b.require.NoErrorf(err, "Failed to initiate withdrawal from %v for %v", from, amount)
	b.require.Equal(types.ReceiptStatusSuccessful, withdrawRcpt.Status, "initiating withdrawal failed")
	return &Withdrawal{
		commonImpl:  commonFromT(b.t),
		bridge:      b,
		initReceipt: withdrawRcpt,
	}
}

// ERC20Deposit performs an ERC20 deposit from L1 to L2
func (b *StandardBridge) ERC20Deposit(l1TokenAddr common.Address, l2TokenAddr common.Address, amount eth.ETH, from *EOA) *Deposit {
	// Use the l1StandardBridge to deposit ERC20 tokens
	depositCall := b.l1StandardBridge.DepositERC20To(l1TokenAddr, l2TokenAddr, from.Address(), amount, 200000, []byte{})
	depositReceipt, err := contractio.Write(depositCall, b.ctx, from.Plan())
	b.require.NoError(err, "Failed to send ERC20 deposit transaction")
	b.require.Equal(types.ReceiptStatusSuccessful, depositReceipt.Status, "ERC20 deposit should succeed")

	// Wait for the deposit to be processed on the L2
	// Find the deposit log to get the L2 deposit transaction
	var l2DepositTx *optypes.DepositTx
	for _, log := range depositReceipt.Logs {
		if l2DepositTx, err = derive.UnmarshalDepositLogEvent(log); err == nil {
			break
		}
	}
	b.require.NotNil(l2DepositTx, "Could not find L2 deposit transaction in logs")

	l2DepositTxHash := l2DepositTx.Hash()

	// Give time for L2CL to include the L2 deposit tx
	sequencingWindowDuration := time.Duration(b.rollupCfg.SeqWindowSize) * b.l1Client.EstimateBlockTime()
	var l2DepositReceipt *optypes.Receipt
	b.require.Eventually(func() bool {
		l2DepositReceipt, err = b.l2Client.TransactionReceipt(b.ctx, l2DepositTxHash)
		return err == nil
	}, sequencingWindowDuration, 500*time.Millisecond, "L2 ERC20 deposit never found")
	b.require.Equal(types.ReceiptStatusSuccessful, l2DepositReceipt.Status, "L2 ERC20 deposit should succeed")

	return &Deposit{
		bridge:    b,
		l1Receipt: depositReceipt,
	}
}

// CreateL2Token creates an L2 token using OptimismMintableERC20Factory and returns the token address
func (b *StandardBridge) CreateL2Token(l1TokenAddr common.Address, name string, symbol string, from *EOA) common.Address {
	factoryContract := bindings.NewBindings[bindings.OptimismMintableERC20Factory](
		bindings.WithTest(b.t),
		bindings.WithClient(b.l2Client),
		bindings.WithTo(predeploys.OptimismMintableERC20FactoryAddr),
	)

	createCall := factoryContract.CreateOptimismMintableERC20(l1TokenAddr, name, symbol)
	createReceipt, err := contractio.Write(createCall, b.ctx, from.Plan())
	b.require.NoError(err, "Failed to create L2 token")
	b.require.Equal(types.ReceiptStatusSuccessful, createReceipt.Status, "L2 token creation should succeed")

	// Extract L2 token address from logs
	l2TokenAddress := b.extractL2TokenFromLogs(createReceipt)
	b.log.Info("Created L2 token", "l1Token", l1TokenAddr, "l2Token", l2TokenAddress, "name", name, "symbol", symbol)
	return l2TokenAddress
}

// extractL2TokenFromLogs extracts the L2 token address from OptimismMintableERC20Created event
func (b *StandardBridge) extractL2TokenFromLogs(receipt *types.Receipt) common.Address {
	// Look for the OptimismMintableERC20Created event
	for _, log := range receipt.Logs {
		if log.Address == predeploys.OptimismMintableERC20FactoryAddr && len(log.Topics) > 2 {
			// The token address is in the indexed topics
			return common.HexToAddress(log.Topics[2].Hex())
		}
	}
	b.require.Fail("Failed to find L2 token address from events")
	return common.Address{} // Never reached
}

type Withdrawal struct {
	commonImpl
	bridge      *StandardBridge
	initReceipt *types.Receipt

	proveParams     ProvenWithdrawalParameters
	proveReceipt    *types.Receipt
	finalizeReceipt *types.Receipt
}

func (w *Withdrawal) InitiateGasCost() eth.ETH {
	return w.bridge.gasCost(w.initReceipt, w.bridge.l2Client)
}

func (w *Withdrawal) ProveGasCost() eth.ETH {
	w.require.NotNil(w.proveReceipt, "Must have proven withdrawal before calculating gas cost")
	return w.bridge.gasCost(w.proveReceipt, w.bridge.l1Client.EthClient())
}

func (w *Withdrawal) ProvenDisputeGameIndex() *big.Int {
	w.require.NotNil(w.proveReceipt, "Must have proven withdrawal before reading dispute game index")
	w.require.NotNil(w.proveParams.DisputeGameIndex, "Proven withdrawal is missing dispute game index")
	return new(big.Int).Set(w.proveParams.DisputeGameIndex)
}

func (w *Withdrawal) FinalizeGasCost() eth.ETH {
	w.require.NotNil(w.finalizeReceipt, "Must have finalized withdrawal before calculating gas cost")
	return w.bridge.gasCost(w.finalizeReceipt, w.bridge.l1Client.EthClient())
}

func (w *Withdrawal) InitiateBlockHash() common.Hash {
	return w.initReceipt.BlockHash
}

func (w *Withdrawal) InitiateTxHash() common.Hash {
	return w.initReceipt.TxHash
}

func (w *Withdrawal) Prove(user *EOA) {
	var params ProvenWithdrawalParameters

	w.t.Log("proveWithdrawal: proving withdrawal...")
	params = w.proveWithdrawalParameters()
	tx := bindings.WithdrawalTransaction{
		Nonce:    params.Nonce,
		Sender:   params.Sender,
		Target:   params.Target,
		Value:    params.Value,
		GasLimit: params.GasLimit,
		Data:     params.Data,
	}

	// OptimismPortal2.proveWithdrawalTransaction reverts with
	// OptimismPortal_InvalidProofTimestamp (selector 0xb4caa4e5) when
	// block.timestamp <= disputeGameProxy.createdAt(). estimateGas evaluates
	// against the current L1 head, so any attempt before the head advances past
	// the game's createdAt is guaranteed to revert. Wait for that precondition
	// explicitly instead of burning the prove-tx retry budget on guaranteed-revert
	// estimateGas calls; under CI load L1 block production can stall and the
	// prior retry-only approach exhausted its 30s budget before the head moved
	// (#19963).
	gameContract := bindings.NewBindings[bindings.FaultDisputeGame](
		bindings.WithClient(w.bridge.l1Client.EthClient()),
		bindings.WithTo(params.DisputeGameAddress),
		bindings.WithTest(w.t))
	gameCreatedAt, err := contractio.Read(gameContract.CreatedAt(), w.ctx)
	w.require.NoError(err, "failed to read dispute game createdAt")
	w.require.Eventuallyf(func() bool {
		head, err := w.bridge.l1Client.EthClient().InfoByLabel(w.ctx, eth.Unsafe)
		if err != nil {
			return false
		}
		return head.Time() > gameCreatedAt
	}, 60*time.Second, 500*time.Millisecond, "L1 head did not advance past dispute game createdAt %d", gameCreatedAt)

	call := w.bridge.l1Portal.ProveWithdrawalTransaction(tx, params.DisputeGameIndex, params.OutputRootProof, params.WithdrawalProof)
	w.require.Eventually(func() bool {
		proveReceipt, err := contractio.Write(call, w.ctx, user.Plan())
		if err != nil {
			w.log.Error("Failed to send prove transaction", "err", err)
			return false
		}
		w.require.Equal(types.ReceiptStatusSuccessful, proveReceipt.Status, "prove withdrawal was not successful")
		w.require.Equal(2, len(proveReceipt.Logs)) // emit WithdrawalProven, WithdrawalProvenExtension1

		w.proveParams = params
		w.proveReceipt = proveReceipt
		return true
	}, 30*time.Second, 1*time.Second, "Sending prove transaction")
}

func (w *Withdrawal) FaultProofProveParams(advanceTime ...func(time.Duration)) withdrawals.ProvenWithdrawalParameters {
	var params withdrawals.ProvenWithdrawalParameters
	var lastErr error
	w.require.Eventuallyf(func() bool {
		params, lastErr = w.bridge.faultProofProveParams(w)
		if lastErr == nil {
			return true
		}
		w.log.Warn("Failed to build fault proof withdrawal parameters", "err", lastErr)
		if len(advanceTime) != 0 {
			advanceTime[0](2 * time.Second)
		}
		return false
	}, 90*time.Second, time.Second, "failed to build fault proof withdrawal parameters")
	return params
}

func (b *StandardBridge) faultProofProveParams(withdrawal *Withdrawal) (withdrawals.ProvenWithdrawalParameters, error) {
	l1Client, err := ethclient.DialContext(b.ctx, b.l1Client.Escape().UserRPC())
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, fmt.Errorf("failed to dial L1 RPC: %w", err)
	}
	defer l1Client.Close()

	l2RPC, err := rpc.DialContext(b.ctx, b.l2EL.Escape().UserRPC())
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, fmt.Errorf("failed to dial L2 RPC: %w", err)
	}
	defer l2RPC.Close()

	proofClient := gethclient.New(l2RPC)
	l2Client := ethclient.NewClient(l2RPC)
	defer l2Client.Close()

	portal, err := bindingspreview.NewOptimismPortal2(b.l1PortalAddr, l1Client)
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, fmt.Errorf("failed to bind OptimismPortal2: %w", err)
	}
	factoryAddr, err := portal.DisputeGameFactory(&bind.CallOpts{Context: b.ctx})
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, fmt.Errorf("failed to read dispute game factory: %w", err)
	}
	factory, err := nodebindings.NewDisputeGameFactoryCaller(factoryAddr, l1Client)
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, fmt.Errorf("failed to bind dispute game factory: %w", err)
	}

	params, err := withdrawals.ProveWithdrawalParametersFaultProofs(
		b.ctx,
		proofClient,
		l2Client,
		l2Client,
		withdrawal.InitiateTxHash(),
		factory,
		&portal.OptimismPortal2Caller,
	)
	if err != nil {
		return withdrawals.ProvenWithdrawalParameters{}, err
	}
	return params, nil
}

func (b *StandardBridge) ProveWithFaultProofParams(user *EOA, params withdrawals.ProvenWithdrawalParameters) *types.Receipt {
	gameInfo, err := contractio.Read(b.disputeGameFactory.GameAtIndex(params.L2OutputIndex), b.ctx)
	b.require.NoErrorf(err, "failed to read dispute game %v", params.L2OutputIndex)
	b.require.Eventuallyf(func() bool {
		head, err := b.l1Client.EthClient().InfoByLabel(b.ctx, eth.Unsafe)
		return err == nil && head.Time() > gameInfo.Timestamp
	}, 60*time.Second, 500*time.Millisecond, "L1 head did not advance past dispute game createdAt %d", gameInfo.Timestamp)

	tx := bindings.WithdrawalTransaction{
		Nonce:    params.Nonce,
		Sender:   params.Sender,
		Target:   params.Target,
		Value:    params.Value,
		GasLimit: params.GasLimit,
		Data:     params.Data,
	}
	proof := bindings.OutputRootProof{
		Version:                  params.OutputRootProof.Version,
		StateRoot:                params.OutputRootProof.StateRoot,
		MessagePasserStorageRoot: params.OutputRootProof.MessagePasserStorageRoot,
		LatestBlockhash:          params.OutputRootProof.LatestBlockhash,
	}
	call := b.l1Portal.ProveWithdrawalTransaction(tx, params.L2OutputIndex, proof, params.WithdrawalProof)
	var receipt *types.Receipt
	b.require.Eventually(func() bool {
		receipt, err = contractio.Write(call, b.ctx, user.Plan())
		if err != nil {
			b.log.Error("Failed to send prove transaction", "err", err)
			return false
		}
		return true
	}, 30*time.Second, time.Second, "Sending prove transaction with fault proof params")
	b.require.Equal(types.ReceiptStatusSuccessful, receipt.Status, "prove withdrawal was not successful")
	return receipt
}

// proveWithdrawalParameters waits for a covering dispute game and builds the proof with the same
// helpers as the op-chain-ops withdrawal tool, so the acceptance tests exercise that path for every
// respected game type.
func (w *Withdrawal) proveWithdrawalParameters() ProvenWithdrawalParameters {
	b := w.bridge
	l1Client, err := ethclient.DialContext(w.ctx, b.l1Client.Escape().UserRPC())
	w.require.NoError(err, "failed to dial L1 RPC")
	defer l1Client.Close()

	factoryAddr, err := contractio.Read(b.l1Portal.DisputeGameFactoryAddr(), w.ctx)
	w.require.NoError(err, "failed to read dispute game factory address")
	l2BlockNum := w.initReceipt.BlockNumber
	l2BlockTimestamp := b.rollupCfg.TimestampForBlock(bigs.Uint64Strict(l2BlockNum))
	_, err = wait.ForGamePublished(w.ctx, l1Client, b.l1PortalAddr, factoryAddr, l2BlockNum, l2BlockTimestamp)
	w.require.NoErrorf(err, "no dispute game covering l2 block %v was published", l2BlockNum)

	params := w.FaultProofProveParams()
	game, err := contractio.Read(b.disputeGameFactory.GameAtIndex(params.L2OutputIndex), w.ctx)
	w.require.NoErrorf(err, "failed to read dispute game %v", params.L2OutputIndex)
	return ProvenWithdrawalParameters{
		Nonce:              params.Nonce,
		Sender:             params.Sender,
		Target:             params.Target,
		Value:              params.Value,
		GasLimit:           params.GasLimit,
		DisputeGameAddress: game.Proxy,
		DisputeGameIndex:   params.L2OutputIndex,
		Data:               params.Data,
		OutputRootProof: bindings.OutputRootProof{
			Version:                  params.OutputRootProof.Version,
			StateRoot:                params.OutputRootProof.StateRoot,
			MessagePasserStorageRoot: params.OutputRootProof.MessagePasserStorageRoot,
			LatestBlockhash:          params.OutputRootProof.LatestBlockhash,
		},
		WithdrawalProof: params.WithdrawalProof,
	}
}

func (w *Withdrawal) Finalize(user *EOA) {
	wd := crossdomain.Withdrawal{
		Nonce:    w.proveParams.Nonce,
		Sender:   &w.proveParams.Sender,
		Target:   &w.proveParams.Target,
		Value:    w.proveParams.Value,
		GasLimit: w.proveParams.GasLimit,
		Data:     w.proveParams.Data,
	}

	// Finalize withdrawal
	w.log.Info("FinalizeWithdrawal: finalizing withdrawal...")
	var finalizeReceipt *types.Receipt
	var err error
	// Retry as the air gap delay needs to have expired at the head block timestamp for estimateGas to work
	w.require.Eventually(func() bool {
		finalizeReceipt, err = contractio.Write(w.bridge.l1Portal.FinalizeWithdrawalTransaction(wd.WithdrawalTransaction()), w.ctx, user.Plan())
		if err != nil {
			return false
		}
		w.finalizeReceipt = finalizeReceipt
		return types.ReceiptStatusSuccessful == finalizeReceipt.Status
	}, 60*time.Second, 100*time.Millisecond, "finalize withdrawal failed")
}

func (w *Withdrawal) WaitForDisputeGameResolved() {
	w.require.NotNil(w.proveReceipt, "Must have proven withdrawal first")

	gameContract := bindings.NewBindings[bindings.FaultDisputeGame](
		bindings.WithClient(w.bridge.l1Client.EthClient()),
		bindings.WithTo(w.proveParams.DisputeGameAddress),
		bindings.WithTest(w.t))
	w.require.Eventually(func() bool {
		status, err := contractio.Read(gameContract.Status(), w.ctx)
		w.require.NoError(err, "failed to get game status")
		w.log.Info("Waiting for dispute game to resolve", "currentStatus", status)
		return gameTypes.GameStatus(status) == gameTypes.GameStatusDefenderWon
	}, 60*time.Second, 100*time.Millisecond, "wait for dispute game resolved")
}

func (b *StandardBridge) gasCost(rcpt *types.Receipt, client apis.EthClient) eth.ETH {
	var blockTimestamp *uint64
	if hasOperatorFee(rcpt) {
		b.require.NotNil(client, "client is required to resolve operator fee timestamp")
		blockTimestamp = b.receiptTimestamp(rcpt, client)
	}
	return gasCost(rcpt, b.rollupCfg, blockTimestamp)
}

func hasOperatorFee(rcpt *types.Receipt) bool {
	return rcpt.OperatorFeeConstant != nil && rcpt.OperatorFeeScalar != nil
}

func (b *StandardBridge) receiptTimestamp(rcpt *types.Receipt, client apis.EthClient) *uint64 {
	b.require.NotNil(rcpt.BlockNumber, "receipt missing block number")
	blockInfo, err := client.InfoByNumber(b.ctx, bigs.Uint64Strict(rcpt.BlockNumber))
	b.require.NoError(err, "failed to fetch block info for receipt")
	ts := blockInfo.Time()
	return &ts
}

func gasCost(rcpt *types.Receipt, rollupCfg *rollup.Config, blockTimestamp *uint64) eth.ETH {
	cost := eth.WeiBig(new(big.Int).Mul(new(big.Int).SetUint64(rcpt.GasUsed), rcpt.EffectiveGasPrice))
	if rcpt.L1Fee != nil {
		cost = cost.Add(eth.WeiBig(rcpt.L1Fee))
	}
	if hasOperatorFee(rcpt) {
		if rollupCfg == nil {
			panic("rollup config is required to compute operator fee")
		}
		if blockTimestamp == nil {
			panic("block timestamp is required to compute operator fee")
		}
		operatorCost := new(big.Int).SetUint64(rcpt.GasUsed)
		operatorCost.Mul(operatorCost, new(big.Int).SetUint64(*rcpt.OperatorFeeScalar))
		if rollupCfg.IsJovian(*blockTimestamp) {
			operatorCost.Mul(operatorCost, big.NewInt(100))
		} else {
			operatorCost.Div(operatorCost, big.NewInt(1_000_000))
		}
		operatorCost.Add(operatorCost, new(big.Int).SetUint64(*rcpt.OperatorFeeConstant))
		cost = cost.Add(eth.WeiBig(operatorCost))
	}
	return cost
}

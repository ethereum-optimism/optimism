package dsl

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
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
		gameType == gameTypes.SuperCannonKonaGameType
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
	l2DepositTxHash := types.NewTx(l2DepositTx).Hash()
	// Give time for L2CL to include the L2 deposit tx
	var l2DepositReceipt *types.Receipt
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
	var l2DepositTx *types.DepositTx
	for _, log := range depositReceipt.Logs {
		if l2DepositTx, err = derive.UnmarshalDepositLogEvent(log); err == nil {
			break
		}
	}
	b.require.NotNil(l2DepositTx, "Could not find L2 deposit transaction in logs")

	l2DepositTxHash := types.NewTx(l2DepositTx).Hash()

	// Give time for L2CL to include the L2 deposit tx
	sequencingWindowDuration := time.Duration(b.rollupCfg.SeqWindowSize) * b.l1Client.EstimateBlockTime()
	var l2DepositReceipt *types.Receipt
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

type disputeGame struct {
	Index          *big.Int
	Address        common.Address
	L2BlockNumber  uint64
	SequenceNumber uint64
	OutputRoot     common.Hash
	UsesSuperRoots bool
}

// forGamePublished waits until the earliest game that covers the given l2BlockNumber is published on L1.
// Note that the l2 block number is passed even for super games. Conversion to timestamp is done automatically
// when required by the respected game type
func (b *StandardBridge) forGamePublished(l2BlockNumber *big.Int) disputeGame {
	return b.waitForCoveringGames(l2BlockNumber, 1)[0]
}

func (b *StandardBridge) waitForCoveringGames(l2BlockNumber *big.Int, count int) []disputeGame {
	b.require.Positive(count, "expected covering game count must be positive")

	respectedGameType := b.RespectedGameType()
	minSequence := bigs.Uint64Strict(l2BlockNumber)
	superRootsActive := b.UsesSuperRoots()
	if superRootsActive {
		minSequence = b.rollupCfg.TimestampForBlock(minSequence)
	}

	var games []disputeGame
	b.require.Eventuallyf(func() bool {
		var err error
		games, err = b.findCoveringGames(respectedGameType, new(big.Int).SetUint64(minSequence), superRootsActive)
		if err != nil {
			b.log.Warn("No covering game of required type found", "err", err)
			return false
		}
		if len(games) < count {
			b.log.Info("Waiting for covering games", "found", len(games), "expected", count, "minSequence", minSequence)
			return false
		}
		b.log.Info("Found covering games", "count", len(games), "earliestIndex", games[0].Index, "earliestSeqNum", games[0].SequenceNumber, "earliestBlock", games[0].L2BlockNumber)
		return true
	}, 90*time.Second, 100*time.Millisecond, "did not find %d games of type %v at or after l2 sequence number %v", count, respectedGameType, minSequence)

	return games
}

func (b *StandardBridge) findCoveringGames(gameType uint32, minSequence *big.Int, superRootsActive bool) ([]disputeGame, error) {
	gameCount, err := contractio.Read(b.disputeGameFactory.GameCount(), b.ctx)
	b.require.NoError(err, "Failed to read game count")
	if gameCount.Cmp(common.Big0) == 0 {
		return nil, errors.New("no games")
	}

	type candidate struct {
		index      *big.Int
		sequence   *big.Int
		outputRoot common.Hash
	}
	var candidates []candidate
	l2ChainID := b.rollupCfg.L2ChainID
	searchStart := new(big.Int).Sub(gameCount, common.Big1)
	for searchStart.Sign() >= 0 {
		games, err := contractio.Read(b.disputeGameFactory.FindLatestGames(gameType, searchStart, big.NewInt(32)), b.ctx)
		b.require.NoErrorf(err, "Failed to find latest games for %v", gameType)
		if len(games) == 0 {
			break
		}
		for _, game := range games {
			sequence, outputRoot, ok, err := bridgeGameSequenceAndOutputRoot(game, gameTypes.GameType(gameType), l2ChainID)
			if err != nil {
				return nil, fmt.Errorf("failed to decode game %v: %w", game.Index, err)
			}
			if ok && sequence.Cmp(minSequence) >= 0 {
				candidates = append(candidates, candidate{
					index:      game.Index,
					sequence:   sequence,
					outputRoot: outputRoot,
				})
			}
			searchStart = new(big.Int).Sub(game.Index, common.Big1)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no covering game found for sequence %v", minSequence)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].index.Cmp(candidates[j].index) < 0
	})

	coveringGames := make([]disputeGame, 0, len(candidates))
	for _, selected := range candidates {
		gameAtIndex, err := contractio.Read(b.disputeGameFactory.GameAtIndex(selected.index), b.ctx)
		b.require.NoErrorf(err, "Failed to get game at index %v", selected.index)
		gameBlockNum := bigs.Uint64Strict(selected.sequence)
		if superRootsActive {
			blockNum, err := b.rollupCfg.TargetBlockNumber(gameBlockNum)
			b.require.NoError(err, "Failed to convert game timestamp to block number")
			gameBlockNum = blockNum
		}
		coveringGames = append(coveringGames, disputeGame{
			Index:          selected.index,
			Address:        gameAtIndex.Proxy,
			L2BlockNumber:  gameBlockNum,
			SequenceNumber: bigs.Uint64Strict(selected.sequence),
			OutputRoot:     selected.outputRoot,
			UsesSuperRoots: superRootsActive,
		})
	}
	return coveringGames, nil
}

func bridgeGameSequenceAndOutputRoot(game bindings.GameSearchResult, gameType gameTypes.GameType, l2ChainID *big.Int) (*big.Int, common.Hash, bool, error) {
	switch gameType {
	case gameTypes.CannonKonaGameType, gameTypes.PermissionedGameType:
		if len(game.ExtraData) < 32 {
			return nil, common.Hash{}, false, fmt.Errorf("legacy game extra data is %d bytes, need at least 32", len(game.ExtraData))
		}
		return new(big.Int).SetBytes(game.ExtraData[:32]), game.RootClaim, true, nil
	case gameTypes.SuperCannonKonaGameType, gameTypes.SuperPermissionedGameType:
		return bridgeSuperRootChainOutput(game.ExtraData, l2ChainID)
	default:
		return nil, common.Hash{}, false, fmt.Errorf("unsupported game type: %v", gameType)
	}
}

func bridgeSuperRootChainOutput(extraData []byte, l2ChainID *big.Int) (*big.Int, common.Hash, bool, error) {
	if l2ChainID == nil {
		return nil, common.Hash{}, false, errors.New("l2 chain id is required for super root games")
	}
	super, err := eth.UnmarshalSuperRoot(extraData)
	if err != nil {
		return nil, common.Hash{}, false, fmt.Errorf("failed to decode super root: %w", err)
	}
	superV1, ok := super.(*eth.SuperV1)
	if !ok {
		return nil, common.Hash{}, false, fmt.Errorf("unsupported super root type %T", super)
	}
	targetChainID := eth.ChainIDFromBig(l2ChainID)
	sequence := new(big.Int).SetUint64(superV1.Timestamp)
	for _, chain := range superV1.Chains {
		if chain.ChainID.Cmp(targetChainID) == 0 {
			return sequence, common.Hash(chain.Output), true, nil
		}
	}
	return sequence, common.Hash{}, false, nil
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

// ProveWithdrawalParameters calls ProveWithdrawalParametersForBlock with the most recent L2 output after the latest game.
// Ported from op-node/withdrawals/utils.go to fit in the op-devstack
func (w *Withdrawal) proveWithdrawalParameters() ProvenWithdrawalParameters {
	// Wait for a suitable game to be published
	latestGame := w.bridge.forGamePublished(w.initReceipt.BlockNumber)

	// Fetch the block header from the L2 node
	l2Header, err := w.bridge.l2Client.InfoByNumber(w.ctx, latestGame.L2BlockNumber)
	w.require.NoErrorf(err, "failed to fetch block header %v", latestGame.L2BlockNumber)

	ev, err := withdrawals.ParseMessagePassed(w.initReceipt)
	w.require.NoError(err, "failed to parse message passed receipt")
	return w.proveWithdrawalParametersForEvent(ev, l2Header, latestGame)
}

// proveWithdrawalParametersForEvent queries L1 to generate all withdrawal parameters and proof necessary to prove a withdrawal on L1.
// The l2Header provided is very important. It should be a block for which there is a submitted output in the L2 Output Oracle
// contract. If not, the withdrawal will fail as it the storage proof cannot be verified if there is no submitted state root.
// Ported from op-node/withdrawals/utils.go to fit in the op-devstack, using op-service ethclient
func (w *Withdrawal) proveWithdrawalParametersForEvent(ev *nodebindings.L2ToL1MessagePasserMessagePassed, l2Header eth.BlockInfo, disputeGame disputeGame) ProvenWithdrawalParameters {
	// Generate then verify the withdrawal proof
	withdrawalHash, err := withdrawals.WithdrawalHash(ev)
	w.require.NoErrorf(err, "failed to calculate hash for withdrawal %v", ev)
	w.require.Equal(withdrawalHash[:], ev.WithdrawalHash[:], "computed withdrawal hash incorrectly")
	slot := withdrawals.StorageSlotOfWithdrawalHash(withdrawalHash)

	// op-reth persists state asynchronously, so eth_getProof can briefly fail
	// after the dispute game for the block exists. Retry until it succeeds.
	blockTag := hexutil.Uint64(l2Header.NumberU64()).String()
	var p *eth.AccountResult
	w.require.Eventuallyf(func() bool {
		var err error
		p, err = w.bridge.l2Client.GetProof(w.ctx, predeploys.L2ToL1MessagePasserAddr, []common.Hash{slot}, blockTag)
		return err == nil
	}, 60*time.Second, 500*time.Millisecond, "failed to fetch proof for withdrawal at block %d: %v", l2Header.NumberU64(), ev)
	w.require.Len(p.StorageProof, 1, "invalid amount of storage proofs")

	err = verifyProof(l2Header.Root(), p)
	w.require.NoErrorf(err, "failed to verify proof for withdrawal")

	// Encode it as expected by the contract
	trieNodes := make([][]byte, len(p.StorageProof[0].Proof))
	for i, s := range p.StorageProof[0].Proof {
		trieNodes[i] = s
	}

	params := ProvenWithdrawalParameters{
		Nonce:              ev.Nonce,
		Sender:             ev.Sender,
		Target:             ev.Target,
		Value:              ev.Value,
		GasLimit:           ev.GasLimit,
		DisputeGameAddress: disputeGame.Address,
		DisputeGameIndex:   disputeGame.Index,
		Data:               ev.Data,
		OutputRootProof: bindings.OutputRootProof{
			Version:                  [32]byte{}, // Empty for version 1
			StateRoot:                l2Header.Root(),
			MessagePasserStorageRoot: *l2Header.WithdrawalsRoot(),
			LatestBlockhash:          l2Header.Hash(),
		},
		WithdrawalProof: trieNodes,
	}
	outputRoot := eth.OutputRoot(&eth.OutputV0{
		StateRoot:                eth.Bytes32(params.OutputRootProof.StateRoot),
		MessagePasserStorageRoot: eth.Bytes32(params.OutputRootProof.MessagePasserStorageRoot),
		BlockHash:                common.Hash(params.OutputRootProof.LatestBlockhash),
	})
	w.require.Equalf(disputeGame.OutputRoot, common.Hash(outputRoot),
		"computed output root must match dispute game root claim for game index %v", disputeGame.Index)
	return params
}

// Ported from op-node/withdrawals/proof.go to fit in the op-devstack, using op-service proof types
func verifyProof(stateRoot common.Hash, proof *eth.AccountResult) error {
	balance, overflow := uint256.FromBig(proof.Balance.ToInt())
	if overflow {
		return fmt.Errorf("proof balance overflows uint256: %d", proof.Balance.ToInt())
	}
	proofHex := []string{}
	for _, p := range proof.AccountProof {
		proofHex = append(proofHex, hex.EncodeToString(p))
	}
	err := withdrawals.VerifyAccountProof(
		stateRoot,
		proof.Address,
		types.StateAccount{
			Nonce:    uint64(proof.Nonce),
			Balance:  balance,
			Root:     proof.StorageHash,
			CodeHash: proof.CodeHash[:],
		},
		proofHex,
	)
	if err != nil {
		return fmt.Errorf("failed to validate account: %w", err)
	}
	for i, storageProof := range proof.StorageProof {
		proofHex := []string{}
		for _, p := range storageProof.Proof {
			proofHex = append(proofHex, hex.EncodeToString(p))
		}
		convertedProof := gethclient.StorageResult{
			Key:   storageProof.Key.String(),
			Value: storageProof.Value.ToInt(),
			Proof: proofHex,
		}
		err = withdrawals.VerifyStorageProof(proof.StorageHash, convertedProof)
		if err != nil {
			return fmt.Errorf("failed to validate storage proof %d: %w", i, err)
		}
	}
	return nil
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

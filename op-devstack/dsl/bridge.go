package dsl

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	nodebindings "github.com/ethereum-optimism/optimism/op-node/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
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
	SuperRootProof     *bindings.SuperRootProof // Only set for games using super roots
	OutputRootIndex    *big.Int                 // Only set for games using super roots
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

	l1Client         apis.EthClient
	l2Client         apis.EthClient
	supervisorClient apis.SupervisorQueryAPI

	// L1 bridge contract
	l1StandardBridge bindings.L1StandardBridge
}

func NewStandardBridge(t devtest.T, l2Network *L2Network, supervisor *Supervisor, l1EL *L1ELNode) *StandardBridge {
	l1Client := l1EL.EthClient()
	l1PortalAddr := l2Network.DepositContractAddr()
	l1Portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(l1Client),
		bindings.WithTo(l1PortalAddr),
		bindings.WithTest(t))
	l2Client := l2Network.inner.L2ELNode(match.FirstL2EL).EthClient()
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

	var supervisorClient apis.SupervisorQueryAPI
	if supervisor != nil {
		supervisorClient = supervisor.inner.QueryAPI()
	}
	return &StandardBridge{
		commonImpl:          commonFromT(t),
		l1PortalAddr:        l1PortalAddr,
		l1Portal:            l1Portal,
		l2tol1MessagePasser: l2tol1MessagePasser,
		disputeGameFactory:  disputeGameFactory,
		rollupCfg:           l2Network.inner.RollupConfig(),

		l1Client:         l1Client,
		l2Client:         l2Client,
		supervisorClient: supervisorClient,
		l1StandardBridge: l1StandardBridge,
	}
}

func (b *StandardBridge) GameResolutionDelay() time.Duration {
	gameType := b.RespectedGameType()
	gameImplAddr, err := contractio.Read(b.disputeGameFactory.GameImpls(gameType), b.ctx)
	b.require.NoErrorf(err, "failed to get implementation for game type %v", gameType)
	game := bindings.NewBindings[bindings.FaultDisputeGame](bindings.WithClient(b.l1Client), bindings.WithTo(gameImplAddr), bindings.WithTest(b.t))
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

func (b *StandardBridge) UsesSuperRoots() bool {
	superRootsActive, err := contractio.Read(b.l1Portal.SuperRootsActive(), b.ctx)
	b.require.NoError(err, "Failed to read super roots active")
	return superRootsActive
}

type Deposit struct {
	l1Receipt *types.Receipt
}

func (d Deposit) GasCost() eth.ETH {
	return gasCost(d.l1Receipt)
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
	var l2DepositReceipt *types.Receipt
	b.require.Eventually(func() bool {
		l2DepositReceipt, err = b.l2Client.TransactionReceipt(b.ctx, l2DepositTxHash)
		return err == nil
	}, 60*time.Second, 500*time.Millisecond, "L2 ERC20 deposit never found")
	b.require.Equal(types.ReceiptStatusSuccessful, l2DepositReceipt.Status, "L2 ERC20 deposit should succeed")

	return &Deposit{
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
	UsesSuperRoots bool
}

// forGamePublished waits until a game is published on L1 for the given l2BlockNumber
// Note that the l2 block number is passed even for super games. Conversion to timestamp is done automatically
// when required by the respected game type
func (b *StandardBridge) forGamePublished(l2BlockNumber *big.Int) disputeGame {
	respectedGameType := b.RespectedGameType()
	l2SequenceNumber := l2BlockNumber.Uint64()
	superRootsActive := b.UsesSuperRoots()
	if superRootsActive {
		l2SequenceNumber = b.rollupCfg.TimestampForBlock(l2SequenceNumber)
	}

	var game bindings.DisputeGame
	var gameSeqNum uint64
	var gameIndex *big.Int
	b.require.Eventuallyf(func() bool {
		var err error
		game, gameIndex, err = b.findLatestGame(respectedGameType)
		if err != nil {
			b.log.Warn("No game of required type found", "err", err)
			return false
		}
		gameContract := bindings.NewBindings[bindings.FaultDisputeGame](
			bindings.WithClient(b.l1Client),
			bindings.WithTo(game.Proxy),
			bindings.WithTest(b.t))
		seqNum, err := contractio.Read(gameContract.L2SequenceNumber(), b.ctx)
		b.require.NoError(err, "Failed to read sequence number")
		gameSeqNum = seqNum.Uint64()
		b.log.Info("Found latest game", "index", gameIndex, "seqNum", gameSeqNum)
		return gameSeqNum >= l2SequenceNumber
	}, 90*time.Second, 100*time.Millisecond, "did not find a game of type %v at or after l2 sequence number %v", respectedGameType, l2SequenceNumber)

	gameBlockNum := gameSeqNum
	if superRootsActive {
		blockNum, err := b.rollupCfg.TargetBlockNumber(gameSeqNum)
		b.require.NoError(err, "Failed to convert game timestamp to block number")
		gameBlockNum = blockNum
	}
	return disputeGame{
		Index:          gameIndex,
		Address:        game.Proxy,
		L2BlockNumber:  gameBlockNum,
		SequenceNumber: gameSeqNum,
		UsesSuperRoots: superRootsActive,
	}
}

// findLatestGame finds the latest game in the DisputeGameFactory contract.
// Ported from op-node/withdrawals/utils.go to fit in the op-devstack, using op-service ethclient
func (b *StandardBridge) findLatestGame(gameType uint32) (bindings.DisputeGame, *big.Int, error) {
	gameCount, err := contractio.Read(b.disputeGameFactory.GameCount(), b.ctx)
	b.require.NoError(err, "Failed to read game count")
	if gameCount.Cmp(common.Big0) == 0 {
		return bindings.DisputeGame{}, nil, errors.New("no games")
	}

	gameIdx := new(big.Int).Sub(gameCount, common.Big1)
	for gameIdx.Cmp(common.Big0) >= 0 {
		latestGame, err := contractio.Read(b.disputeGameFactory.GameAtIndex(gameIdx), b.ctx)
		b.require.NoErrorf(err, "Failed to find latest game for %v", gameType)
		if latestGame.GameType != gameType {
			// Wrong game type, continue searching backwards
			gameIdx = new(big.Int).Sub(gameIdx, common.Big1)
			continue
		}
		return latestGame, gameIdx, nil
	}
	return bindings.DisputeGame{}, nil, errors.New("no suitable games found")
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
	return gasCost(w.initReceipt)
}

func (w *Withdrawal) ProveGasCost() eth.ETH {
	w.require.NotNil(w.proveReceipt, "Must have proven withdrawal before calculating gas cost")
	return gasCost(w.proveReceipt)
}

func (w *Withdrawal) FinalizeGasCost() eth.ETH {
	w.require.NotNil(w.finalizeReceipt, "Must have finalized withdrawal before calculating gas cost")
	return gasCost(w.finalizeReceipt)
}

func (w *Withdrawal) InitiateBlockHash() common.Hash {
	return w.initReceipt.BlockHash
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

	var call bindings.TypedCall[any]
	if params.SuperRootProof == nil {
		call = w.bridge.l1Portal.ProveWithdrawalTransaction(tx, params.DisputeGameIndex, params.OutputRootProof, params.WithdrawalProof)
	} else {
		call = w.bridge.l1Portal.ProveWithdrawalTransactionSuperRoot(tx, params.DisputeGameAddress, params.OutputRootIndex, *params.SuperRootProof, params.OutputRootProof, params.WithdrawalProof)
	}
	// Retry as withdrawals can't be proven in the same block as the game is created.
	// estimateGas works against the current head so we may need to retry until it has progressed enough.
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

	p, err := w.bridge.l2Client.GetProof(w.ctx, predeploys.L2ToL1MessagePasserAddr, []common.Hash{slot}, hexutil.Uint64(l2Header.NumberU64()).String())
	w.require.NoErrorf(err, "failed to fetch proof for withdrawal %v", ev)
	w.require.Len(p.StorageProof, 1, "invalid amount of storage proofs")

	err = verifyProof(l2Header.Root(), p)
	w.require.NoErrorf(err, "failed to verify proof for withdrawal")

	// Encode it as expected by the contract
	trieNodes := make([][]byte, len(p.StorageProof[0].Proof))
	for i, s := range p.StorageProof[0].Proof {
		trieNodes[i] = s
	}

	var superRootProof *bindings.SuperRootProof
	var outputRootIndex *big.Int
	if disputeGame.UsesSuperRoots {
		response, err := w.bridge.supervisorClient.SuperRootAtTimestamp(w.ctx, hexutil.Uint64(disputeGame.SequenceNumber))
		w.require.NoErrorf(err, "failed to fetch superRoot for at timestamp %v", disputeGame.SequenceNumber)
		outputRoots := make([]bindings.OutputRootWithChainID, len(response.Chains))
		for i, chain := range response.Chains {
			outputRoots[i] = bindings.OutputRootWithChainID{
				ChainID: chain.ChainID.ToBig(),
				Root:    chain.Canonical,
			}
			if chain.ChainID == eth.ChainIDFromBig(w.bridge.rollupCfg.L2ChainID) {
				outputRootIndex = big.NewInt(int64(i))
			}
		}
		superRootProof = &bindings.SuperRootProof{
			Version:     [1]byte{response.Version},
			Timestamp:   response.Timestamp,
			OutputRoots: outputRoots,
		}
	}

	return ProvenWithdrawalParameters{
		Nonce:              ev.Nonce,
		Sender:             ev.Sender,
		Target:             ev.Target,
		Value:              ev.Value,
		GasLimit:           ev.GasLimit,
		DisputeGameAddress: disputeGame.Address,
		DisputeGameIndex:   disputeGame.Index,
		Data:               ev.Data,
		SuperRootProof:     superRootProof,
		OutputRootIndex:    outputRootIndex,
		OutputRootProof: bindings.OutputRootProof{
			Version:                  [32]byte{}, // Empty for version 1
			StateRoot:                l2Header.Root(),
			MessagePasserStorageRoot: *l2Header.WithdrawalsRoot(),
			LatestBlockhash:          l2Header.Hash(),
		},
		WithdrawalProof: trieNodes,
	}
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
		bindings.WithClient(w.bridge.l1Client),
		bindings.WithTo(w.proveParams.DisputeGameAddress),
		bindings.WithTest(w.t))
	w.require.Eventually(func() bool {
		status, err := contractio.Read(gameContract.Status(), w.ctx)
		w.require.NoError(err, "failed to get game status")
		w.log.Info("Waiting for dispute game to resolve", "currentStatus", status)
		return gameTypes.GameStatus(status) == gameTypes.GameStatusDefenderWon
	}, 60*time.Second, 100*time.Millisecond, "wait for dispute game resolved")
}

func gasCost(rcpt *types.Receipt) eth.ETH {
	cost := eth.WeiBig(new(big.Int).Mul(new(big.Int).SetUint64(rcpt.GasUsed), rcpt.EffectiveGasPrice))
	if rcpt.L1Fee != nil {
		cost = cost.Add(eth.WeiBig(rcpt.L1Fee))
	}
	if rcpt.OperatorFeeConstant != nil && rcpt.OperatorFeeScalar != nil {
		// https://github.com/ethereum-optimism/op-geth/blob/6005dd53e1b50fe5a3f59764e3e2056a639eff2f/core/types/rollup_cost.go#L244-L247
		// Also see: https://specs.optimism.io/protocol/isthmus/exec-engine.html#operator-operatorCost
		operatorCost := new(big.Int).SetUint64(rcpt.GasUsed)
		operatorCost.Mul(operatorCost, new(big.Int).SetUint64(*rcpt.OperatorFeeScalar))
		operatorCost = operatorCost.Div(operatorCost, big.NewInt(1_000_000))
		operatorCost = operatorCost.Add(operatorCost, new(big.Int).SetUint64(*rcpt.OperatorFeeConstant))
		cost = cost.Add(eth.WeiBig(operatorCost))
	}
	return cost
}

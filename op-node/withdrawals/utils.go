package withdrawals

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var MessagePassedTopic = crypto.Keccak256Hash([]byte("MessagePassed(uint256,address,address,uint256,uint256,bytes,bytes32)"))

type ProofClient interface {
	GetProof(context.Context, common.Address, []string, *big.Int) (*gethclient.AccountResult, error)
}

type ReceiptClient interface {
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
}

type HeaderClient interface {
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
}

type ChainIDClient interface {
	ChainID(context.Context) (*big.Int, error)
}

// FindL2HeaderForTimestamp finds the highest L2 block header with a timestamp at or before targetTimestamp.
func FindL2HeaderForTimestamp(ctx context.Context, l2HeaderCl HeaderClient, targetTimestamp uint64) (*types.Header, error) {
	latestHeader, err := l2HeaderCl.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest l2 header: %w", err)
	}
	if latestHeader == nil {
		return nil, errors.New("latest l2 header was nil")
	}
	if !latestHeader.Number.IsUint64() {
		return nil, fmt.Errorf("latest l2 block number does not fit in uint64: %v", latestHeader.Number)
	}
	if latestHeader.Time < targetTimestamp {
		return nil, fmt.Errorf("latest l2 header timestamp %d is before target timestamp %d", latestHeader.Time, targetTimestamp)
	}

	var result *types.Header
	low := uint64(0)
	high := bigs.Uint64Strict(latestHeader.Number)
	for low <= high {
		mid := low + (high-low)/2
		header, err := l2HeaderCl.HeaderByNumber(ctx, new(big.Int).SetUint64(mid))
		if err != nil {
			return nil, fmt.Errorf("failed to get l2 header %d: %w", mid, err)
		}
		if header == nil {
			return nil, fmt.Errorf("l2 header %d was nil", mid)
		}

		if header.Time <= targetTimestamp {
			result = header
			if mid == high {
				break
			}
			low = mid + 1
			continue
		}
		if mid == 0 {
			break
		}
		high = mid - 1
	}
	if result == nil {
		return nil, fmt.Errorf("no l2 header found at or before target timestamp %d", targetTimestamp)
	}
	return result, nil
}

type withdrawalGameCandidate struct {
	Game       bindings.IDisputeGameFactoryGameSearchResult
	Sequence   *big.Int
	OutputRoot common.Hash
}

// ProvenWithdrawalParameters is the set of parameters to pass to the ProveWithdrawalTransaction
// and FinalizeWithdrawalTransaction functions
type ProvenWithdrawalParameters struct {
	Nonce           *big.Int
	Sender          common.Address
	Target          common.Address
	Value           *big.Int
	GasLimit        *big.Int
	L2OutputIndex   *big.Int
	Data            []byte
	OutputRootProof bindings.TypesOutputRootProof
	WithdrawalProof [][]byte // List of trie nodes to prove L2 storage
}

// ProveWithdrawalParametersFaultProofs generates withdrawal proof parameters using the latest
// dispute game that covers the withdrawal transaction's L2 block.
func ProveWithdrawalParametersFaultProofs(ctx context.Context, proofCl ProofClient, l2ReceiptCl ReceiptClient, l2HeaderCl HeaderClient, txHash common.Hash, disputeGameFactoryContract *bindings.DisputeGameFactoryCaller, optimismPortal2Contract *bindingspreview.OptimismPortal2Caller) (ProvenWithdrawalParameters, error) {
	respectedGameType, err := optimismPortal2Contract.RespectedGameType(&bind.CallOpts{})
	if err != nil {
		return ProvenWithdrawalParameters{}, fmt.Errorf("failed to get respected game type: %w", err)
	}

	receipt, err := l2ReceiptCl.TransactionReceipt(ctx, txHash)
	if err != nil {
		return ProvenWithdrawalParameters{}, fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	ev, err := ParseMessagePassed(receipt)
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}
	withdrawalHeader, err := l2HeaderCl.HeaderByNumber(ctx, receipt.BlockNumber)
	if err != nil {
		return ProvenWithdrawalParameters{}, fmt.Errorf("failed to get withdrawal l2 header: %w", err)
	}
	if withdrawalHeader == nil {
		return ProvenWithdrawalParameters{}, errors.New("withdrawal l2 header was nil")
	}

	respectedGame := gameTypes.GameType(respectedGameType)
	var minSequence *big.Int
	var l2ChainID *big.Int
	switch respectedGame {
	case gameTypes.CannonKonaGameType, gameTypes.CannonGameType, gameTypes.PermissionedGameType, gameTypes.FastGameType:
		minSequence = new(big.Int).Set(receipt.BlockNumber)
	case gameTypes.SuperCannonKonaGameType, gameTypes.SuperPermissionedGameType:
		minSequence = new(big.Int).SetUint64(withdrawalHeader.Time)
		l2ChainID, err = chainID(ctx, l2ReceiptCl, l2HeaderCl)
		if err != nil {
			return ProvenWithdrawalParameters{}, fmt.Errorf("failed to get l2 chain id: %w", err)
		}
	default:
		return ProvenWithdrawalParameters{}, fmt.Errorf("unsupported game type: %v", respectedGameType)
	}

	candidate, err := findLatestWithdrawalGameCandidate(ctx, disputeGameFactoryContract, respectedGame, minSequence, l2ChainID)
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}

	l2Header, err := l2HeaderForGame(ctx, l2HeaderCl, respectedGame, candidate.Sequence)
	if err != nil {
		return ProvenWithdrawalParameters{}, fmt.Errorf("failed to get l2 header for dispute game %v: %w", candidate.Game.Index, err)
	}
	params, err := ProveWithdrawalParametersForEvent(ctx, proofCl, ev, l2Header, candidate.Game.Index)
	if err != nil {
		return ProvenWithdrawalParameters{}, fmt.Errorf("failed to create proof for dispute game %v: %w", candidate.Game.Index, err)
	}
	provenOutputRoot, err := rollup.ComputeL2OutputRoot(&params.OutputRootProof)
	if err != nil {
		return ProvenWithdrawalParameters{}, fmt.Errorf("failed to compute output root for dispute game %v: %w", candidate.Game.Index, err)
	}
	if common.Hash(provenOutputRoot) != candidate.OutputRoot {
		return ProvenWithdrawalParameters{}, fmt.Errorf("dispute game %v output root mismatch: expected %s, got %s", candidate.Game.Index, candidate.OutputRoot, common.Hash(provenOutputRoot))
	}
	return params, nil
}

func chainID(ctx context.Context, clients ...any) (*big.Int, error) {
	for _, cl := range clients {
		if chainIDCl, ok := cl.(ChainIDClient); ok {
			return chainIDCl.ChainID(ctx)
		}
	}
	return nil, errors.New("l2 client does not support ChainID")
}

func l2HeaderForGame(ctx context.Context, l2HeaderCl HeaderClient, gameType gameTypes.GameType, sequence *big.Int) (*types.Header, error) {
	switch gameType {
	case gameTypes.CannonKonaGameType, gameTypes.CannonGameType, gameTypes.PermissionedGameType, gameTypes.FastGameType:
		header, err := l2HeaderCl.HeaderByNumber(ctx, sequence)
		if err != nil {
			return nil, fmt.Errorf("failed to get l2 header %v: %w", sequence, err)
		}
		if header == nil {
			return nil, fmt.Errorf("l2 header %v was nil", sequence)
		}
		return header, nil
	case gameTypes.SuperCannonKonaGameType, gameTypes.SuperPermissionedGameType:
		if !sequence.IsUint64() {
			return nil, fmt.Errorf("l2 sequence number does not fit in uint64: %v", sequence)
		}
		header, err := FindL2HeaderForTimestamp(ctx, l2HeaderCl, bigs.Uint64Strict(sequence))
		if err != nil {
			return nil, fmt.Errorf("failed to find l2 header for timestamp %v: %w", sequence, err)
		}
		return header, nil
	default:
		return nil, fmt.Errorf("unsupported game type: %v", gameType)
	}
}

// findLatestWithdrawalGameCandidate walks backwards through currently published games and returns
// the first game that covers the withdrawal sequence.
func findLatestWithdrawalGameCandidate(ctx context.Context, disputeGameFactoryContract *bindings.DisputeGameFactoryCaller, gameType gameTypes.GameType, minSequence *big.Int, l2ChainID *big.Int) (withdrawalGameCandidate, error) {
	gameCount, err := disputeGameFactoryContract.GameCount(&bind.CallOpts{Context: ctx})
	if err != nil {
		return withdrawalGameCandidate{}, fmt.Errorf("failed to get game count: %w", err)
	}
	if gameCount.Cmp(common.Big0) == 0 {
		return withdrawalGameCandidate{}, errors.New("no games")
	}

	searchStart := new(big.Int).Sub(gameCount, common.Big1)
	batchSize := big.NewInt(32)
	for searchStart.Sign() >= 0 {
		games, err := disputeGameFactoryContract.FindLatestGames(&bind.CallOpts{Context: ctx}, uint32(gameType), searchStart, batchSize)
		if err != nil {
			return withdrawalGameCandidate{}, fmt.Errorf("failed to get latest games: %w", err)
		}
		if len(games) == 0 {
			break
		}
		for _, game := range games {
			sequence, outputRoot, ok, err := gameSequenceAndOutputRoot(game, gameType, l2ChainID)
			if err != nil {
				return withdrawalGameCandidate{}, fmt.Errorf("failed to decode game %v: %w", game.Index, err)
			}
			if ok && sequence.Cmp(minSequence) >= 0 {
				return withdrawalGameCandidate{
					Game:       game,
					Sequence:   sequence,
					OutputRoot: outputRoot,
				}, nil
			}
			searchStart = new(big.Int).Sub(game.Index, common.Big1)
		}
	}
	return withdrawalGameCandidate{}, fmt.Errorf("no dispute game covers withdrawal sequence %v", minSequence)
}

func gameSequenceAndOutputRoot(game bindings.IDisputeGameFactoryGameSearchResult, gameType gameTypes.GameType, l2ChainID *big.Int) (*big.Int, common.Hash, bool, error) {
	switch gameType {
	case gameTypes.CannonKonaGameType, gameTypes.CannonGameType, gameTypes.PermissionedGameType, gameTypes.FastGameType:
		if len(game.ExtraData) < 32 {
			return nil, common.Hash{}, false, fmt.Errorf("legacy game extra data is %d bytes, need at least 32", len(game.ExtraData))
		}
		return new(big.Int).SetBytes(game.ExtraData[:32]), game.RootClaim, true, nil
	case gameTypes.SuperCannonKonaGameType, gameTypes.SuperPermissionedGameType:
		sequence, root, ok, err := superRootChainOutput(game.ExtraData, l2ChainID)
		return sequence, root, ok, err
	default:
		return nil, common.Hash{}, false, fmt.Errorf("unsupported game type: %v", gameType)
	}
}

func superRootChainOutput(extraData []byte, l2ChainID *big.Int) (*big.Int, common.Hash, bool, error) {
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

// ProveWithdrawalParametersForBlock queries L1 & L2 to generate all withdrawal parameters and proof necessary to prove a withdrawal on L1.
// The l2Header provided is very important. It should be a block for which there is a submitted output in the L2 Output Oracle
// contract. If not, the withdrawal will fail as it the storage proof cannot be verified if there is no submitted state root.
func ProveWithdrawalParametersForBlock(ctx context.Context, proofCl ProofClient, l2ReceiptCl ReceiptClient, txHash common.Hash, l2Header *types.Header, l2OutputIndex *big.Int) (ProvenWithdrawalParameters, error) {
	// Transaction receipt
	receipt, err := l2ReceiptCl.TransactionReceipt(ctx, txHash)
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}
	// Parse the receipt
	ev, err := ParseMessagePassed(receipt)
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}
	return ProveWithdrawalParametersForEvent(ctx, proofCl, ev, l2Header, l2OutputIndex)
}

// ProveWithdrawalParametersForEvent queries L1 to generate all withdrawal parameters and proof necessary to prove a withdrawal on L1.
// The l2Header provided is very important. It should be a block for which there is a submitted output in the L2 Output Oracle
// contract. If not, the withdrawal will fail as it the storage proof cannot be verified if there is no submitted state root.
func ProveWithdrawalParametersForEvent(ctx context.Context, proofCl ProofClient, ev *bindings.L2ToL1MessagePasserMessagePassed, l2Header *types.Header, l2OutputIndex *big.Int) (ProvenWithdrawalParameters, error) {
	withdrawalProof, storageRoot, err := GetWithdrawalProof(ctx, proofCl, ev, l2Header)
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}
	return ProvenWithdrawalParameters{
		Nonce:         ev.Nonce,
		Sender:        ev.Sender,
		Target:        ev.Target,
		Value:         ev.Value,
		GasLimit:      ev.GasLimit,
		L2OutputIndex: l2OutputIndex,
		Data:          ev.Data,
		OutputRootProof: bindings.TypesOutputRootProof{
			Version:                  [32]byte{}, // Empty for version 1
			StateRoot:                l2Header.Root,
			MessagePasserStorageRoot: storageRoot,
			LatestBlockhash:          l2Header.Hash(),
		},
		WithdrawalProof: withdrawalProof,
	}, nil
}

// FindLatestGame finds the latest game in the DisputeGameFactory contract for the specified game type.
func FindLatestGameForGameType(ctx context.Context, disputeGameFactoryContract *bindings.DisputeGameFactoryCaller, gameType uint32) (*bindings.IDisputeGameFactoryGameSearchResult, error) {
	gameCount, err := disputeGameFactoryContract.GameCount(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("failed to get game count: %w", err)
	}
	if gameCount.Cmp(common.Big0) == 0 {
		return nil, errors.New("no games")
	}

	searchStart := new(big.Int).Sub(gameCount, common.Big1)
	latestGames, err := disputeGameFactoryContract.FindLatestGames(&bind.CallOpts{Context: ctx}, gameType, searchStart, common.Big1)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest games: %w", err)
	}
	if len(latestGames) == 0 {
		return nil, errors.New("no latest games")
	}

	latestGame := latestGames[0]
	return &latestGame, nil
}

// FindLatestGame finds the latest game in the DisputeGameFactory contract.
func FindLatestGame(ctx context.Context, disputeGameFactoryContract *bindings.DisputeGameFactoryCaller, optimismPortal2Contract *bindingspreview.OptimismPortal2Caller) (*bindings.IDisputeGameFactoryGameSearchResult, error) {
	respectedGameType, err := optimismPortal2Contract.RespectedGameType(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("failed to get respected game type: %w", err)
	}
	return FindLatestGameForGameType(ctx, disputeGameFactoryContract, respectedGameType)
}

// Standard ABI types copied from golang ABI tests
var (
	Uint256Type, _ = abi.NewType("uint256", "", nil)
	BytesType, _   = abi.NewType("bytes", "", nil)
	AddressType, _ = abi.NewType("address", "", nil)
)

// WithdrawalHash computes the hash of the withdrawal that was stored in the L2toL1MessagePasser
// contract state.
// TODO:
//   - I don't like having to use the ABI Generated struct
//   - There should be a better way to run the ABI encoding
//   - These needs to be fuzzed against the solidity
func WithdrawalHash(ev *bindings.L2ToL1MessagePasserMessagePassed) (common.Hash, error) {
	//  abi.encode(nonce, msg.sender, _target, msg.value, _gasLimit, _data)
	args := abi.Arguments{
		{Name: "nonce", Type: Uint256Type},
		{Name: "sender", Type: AddressType},
		{Name: "target", Type: AddressType},
		{Name: "value", Type: Uint256Type},
		{Name: "gasLimit", Type: Uint256Type},
		{Name: "data", Type: BytesType},
	}
	enc, err := args.Pack(ev.Nonce, ev.Sender, ev.Target, ev.Value, ev.GasLimit, ev.Data)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to pack for withdrawal hash: %w", err)
	}
	return crypto.Keccak256Hash(enc), nil
}

// ParseMessagePassed parses MessagePassed events from
// a transaction receipt. It does not support multiple withdrawals
// per receipt.
func ParseMessagePassed(receipt *types.Receipt) (*bindings.L2ToL1MessagePasserMessagePassed, error) {
	events, err := ParseMessagesPassed(receipt)
	if err != nil {
		return nil, err
	}
	return events[0], nil
}

// ParseMessagesPassed parses MessagePassed events from
// a transaction receipt. It supports multiple withdrawals
// per receipt.
func ParseMessagesPassed(receipt *types.Receipt) ([]*bindings.L2ToL1MessagePasserMessagePassed, error) {
	contract, err := bindings.NewL2ToL1MessagePasser(common.Address{}, nil)
	if err != nil {
		return nil, err
	}

	var events []*bindings.L2ToL1MessagePasserMessagePassed
	for _, log := range receipt.Logs {
		if len(log.Topics) == 0 || log.Topics[0] != MessagePassedTopic {
			continue
		}

		ev, err := contract.ParseMessagePassed(*log)
		if err != nil {
			return nil, fmt.Errorf("failed to parse log: %w", err)
		}
		events = append(events, ev)
	}
	if len(events) == 0 {
		return nil, errors.New("unable to find MessagePassed event")
	}
	return events, nil
}

// StorageSlotOfWithdrawalHash determines the storage slot of the L2ToL1MessagePasser contract to look at
// given a WithdrawalHash
func StorageSlotOfWithdrawalHash(hash common.Hash) common.Hash {
	// The withdrawals mapping is the 0th storage slot in the L2ToL1MessagePasser contract.
	// To determine the storage slot, use keccak256(withdrawalHash ++ p)
	// Where p is the 32 byte value of the storage slot and ++ is concatenation
	buf := make([]byte, 64)
	copy(buf, hash[:])
	return crypto.Keccak256Hash(buf)
}

func GetWithdrawalProof(ctx context.Context, proofCl ProofClient, ev *bindings.L2ToL1MessagePasserMessagePassed, l2Header *types.Header) ([][]byte, common.Hash, error) {
	// Generate then verify the withdrawal proof
	withdrawalHash, err := WithdrawalHash(ev)
	if !bytes.Equal(withdrawalHash[:], ev.WithdrawalHash[:]) {
		return nil, common.Hash{}, errors.New("Computed withdrawal hash incorrectly")
	}
	if err != nil {
		return nil, common.Hash{}, err
	}
	slot := StorageSlotOfWithdrawalHash(withdrawalHash)

	p, err := proofCl.GetProof(ctx, predeploys.L2ToL1MessagePasserAddr, []string{slot.String()}, l2Header.Number)
	if err != nil {
		return nil, common.Hash{}, err
	}
	if len(p.StorageProof) != 1 {
		return nil, common.Hash{}, errors.New("invalid amount of storage proofs")
	}

	err = VerifyProof(l2Header.Root, p)
	if err != nil {
		return nil, common.Hash{}, err
	}

	// Encode it as expected by the contract
	trieNodes := make([][]byte, len(p.StorageProof[0].Proof))
	for i, s := range p.StorageProof[0].Proof {
		trieNodes[i] = common.FromHex(s)
	}
	return trieNodes, p.StorageHash, nil
}

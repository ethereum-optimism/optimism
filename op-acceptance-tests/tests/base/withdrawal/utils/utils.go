package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-node/bindings"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	bindingsnew "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

var MessagePassedTopic = crypto.Keccak256Hash([]byte("MessagePassed(uint256,address,address,uint256,uint256,bytes,bytes32)"))

type ProofClient interface {
	GetProof(context.Context, common.Address, []common.Hash, *big.Int) (*gethclient.AccountResult, error)
}

type ReceiptClient interface {
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
}

type HeaderClient interface {
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
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

// ProveWithdrawalParameters calls ProveWithdrawalParametersForBlock with the most recent L2 output after the latest game.
func ProveWithdrawalParameters(t devtest.T, l2Chain *dsl.L2Network, l1Client apis.EthClient, l2Client apis.EthClient, l2WithdrawalReceipt *types.Receipt) (ProvenWithdrawalParameters, error) {
	latestGame, err := FindLatestGame(t, l2Chain, l1Client)
	if err != nil {
		return ProvenWithdrawalParameters{}, fmt.Errorf("failed to find latest game: %w", err)
	}

	l2BlockNumber := new(big.Int).SetBytes(latestGame.ExtraData[0:32])
	l2OutputIndex := latestGame.Index
	// Fetch the block header from the L2 node
	l2Header, err := l2Client.InfoByNumber(t.Ctx(), l2BlockNumber.Uint64())
	if err != nil {
		return ProvenWithdrawalParameters{}, fmt.Errorf("failed to get l2Block: %w", err)
	}
	return ProveWithdrawalParametersForBlock(t, l2Client, l2WithdrawalReceipt, l2Header, l2OutputIndex)
}

// ProveWithdrawalParametersForBlock queries L1 & L2 to generate all withdrawal parameters and proof necessary to prove a withdrawal on L1.
// The l2Header provided is very important. It should be a block for which there is a submitted output in the L2 Output Oracle
// contract. If not, the withdrawal will fail as it the storage proof cannot be verified if there is no submitted state root.
func ProveWithdrawalParametersForBlock(t devtest.T, l2Client apis.EthClient, l2WithdrawalReceipt *types.Receipt, l2Header eth.BlockInfo, l2OutputIndex *big.Int) (ProvenWithdrawalParameters, error) {
	// Transaction receipt
	// Parse the receipt
	ev, err := ParseMessagePassed(l2WithdrawalReceipt)
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}
	return ProveWithdrawalParametersForEvent(t, l2Client, ev, l2Header, l2OutputIndex)
}

// ProveWithdrawalParametersForEvent queries L1 to generate all withdrawal parameters and proof necessary to prove a withdrawal on L1.
// The l2Header provided is very important. It should be a block for which there is a submitted output in the L2 Output Oracle
// contract. If not, the withdrawal will fail as it the storage proof cannot be verified if there is no submitted state root.
func ProveWithdrawalParametersForEvent(t devtest.T, l2Client apis.EthClient, ev *bindings.L2ToL1MessagePasserMessagePassed, l2Header eth.BlockInfo, l2OutputIndex *big.Int) (ProvenWithdrawalParameters, error) {
	// Generate then verify the withdrawal proof
	withdrawalHash, err := WithdrawalHash(ev)
	if !bytes.Equal(withdrawalHash[:], ev.WithdrawalHash[:]) {
		return ProvenWithdrawalParameters{}, errors.New("computed withdrawal hash incorrectly")
	}
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}
	slot := StorageSlotOfWithdrawalHash(withdrawalHash)

	p, err := l2Client.GetProof(t.Ctx(), predeploys.L2ToL1MessagePasserAddr, []common.Hash{slot}, "0x"+fmt.Sprintf("%x", l2Header.NumberU64()))
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}
	if len(p.StorageProof) != 1 {
		return ProvenWithdrawalParameters{}, errors.New("invalid amount of storage proofs")
	}

	err = VerifyProof(l2Header.Root(), p)
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}

	// Encode it as expected by the contract
	trieNodes := make([][]byte, len(p.StorageProof[0].Proof))
	for i, s := range p.StorageProof[0].Proof {
		trieNodes[i] = s
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
			StateRoot:                eth.Bytes32(l2Header.Root()),
			MessagePasserStorageRoot: *l2Header.WithdrawalsRoot(),
			LatestBlockhash:          l2Header.Hash(),
		},
		WithdrawalProof: trieNodes,
	}, nil
}

// FindLatestGame finds the latest game in the DisputeGameFactory contract.
func FindLatestGame(t devtest.T, l2Chain *dsl.L2Network, l1Client apis.EthClient) (bindingsnew.GameSearchResult, error) {
	rollupConfig := l2Chain.Escape().RollupConfig()
	disputeGameFactoryAddr := l2Chain.Escape().Deployment().DisputeGameFactoryProxyAddr()
	optimismPortalAddr := rollupConfig.DepositContractAddress

	portalFactory := bindingsnew.NewOptimismPortal2Factory(bindingsnew.WithClient(l1Client), bindingsnew.WithTo(optimismPortalAddr), bindingsnew.WithTest(t))
	portal := bindingsnew.NewOptimismPortal2(portalFactory)

	disputeGameFactory := bindingsnew.NewDisputeGameFactory(bindingsnew.WithClient(l1Client), bindingsnew.WithTo(disputeGameFactoryAddr), bindingsnew.WithTest(t))
	disputeGame := bindingsnew.NewDisputeGame(disputeGameFactory)

	respectedGameType := contract.Read(portal.RespectedGameType())

	gameCount := contract.Read(disputeGame.GameCount())
	if gameCount.Cmp(common.Big0) == 0 {
		return bindingsnew.GameSearchResult{}, errors.New("no games")
	}

	searchStart := new(big.Int).Sub(gameCount, common.Big1)
	latestGames := contract.Read(disputeGame.FindLatestGames(respectedGameType, searchStart, common.Big1))
	if len(latestGames) == 0 {
		return bindingsnew.GameSearchResult{}, errors.New("no latest games")
	}

	latestGame := latestGames[0]
	return latestGame, nil
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

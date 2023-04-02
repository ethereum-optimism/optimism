package withdrawals

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
)

var MessagePassedTopic = crypto.Keccak256Hash([]byte("MessagePassed(uint256,address,address,uint256,uint256,bytes,bytes32)"))

// WaitForFinalizationPeriod waits until there is OutputProof for an L2 block number larger than the supplied l2BlockNumber
// and that the output is finalized.
// This functions polls and can block for a very long time if used on mainnet.
// This returns the block number to use for the proof generation.
func WaitForFinalizationPeriod(ctx context.Context, client *ethclient.Client, portalAddr common.Address, l2BlockNumber *big.Int) (uint64, error) {
	l2BlockNumber = new(big.Int).Set(l2BlockNumber) // Don't clobber caller owned l2BlockNumber
	opts := &bind.CallOpts{Context: ctx}

	portal, err := bindings.NewOptimismPortalCaller(portalAddr, client)
	if err != nil {
		return 0, err
	}
	l2OOAddress, err := portal.L2ORACLE(opts)
	if err != nil {
		return 0, err
	}
	l2OO, err := bindings.NewL2OutputOracleCaller(l2OOAddress, client)
	if err != nil {
		return 0, err
	}
	submissionInterval, err := l2OO.SUBMISSIONINTERVAL(opts)
	if err != nil {
		return 0, err
	}
	// Convert blockNumber to submission interval boundary
	rem := new(big.Int)
	l2BlockNumber, rem = l2BlockNumber.DivMod(l2BlockNumber, submissionInterval, rem)
	if rem.Cmp(common.Big0) != 0 {
		l2BlockNumber = l2BlockNumber.Add(l2BlockNumber, common.Big1)
	}
	l2BlockNumber = l2BlockNumber.Mul(l2BlockNumber, submissionInterval)

	finalizationPeriod, err := l2OO.FINALIZATIONPERIODSECONDS(opts)
	if err != nil {
		return 0, err
	}

	latest, err := l2OO.LatestBlockNumber(opts)
	if err != nil {
		return 0, err
	}

	// Now poll for the output to be submitted on chain
	var ticker *time.Ticker
	diff := new(big.Int).Sub(l2BlockNumber, latest)
	if diff.Cmp(big.NewInt(10)) > 0 {
		ticker = time.NewTicker(time.Minute)
	} else {
		ticker = time.NewTicker(time.Second)
	}

loop:
	for {
		select {
		case <-ticker.C:
			latest, err = l2OO.LatestBlockNumber(opts)
			if err != nil {
				return 0, err
			}
			// Already passed the submitted block (likely just equals rather than >= here).
			if latest.Cmp(l2BlockNumber) >= 0 {
				break loop
			}
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}

	// Now wait for it to be finalized
	output, err := l2OO.GetL2OutputAfter(opts, l2BlockNumber)
	if err != nil {
		return 0, err
	}
	if output.OutputRoot == [32]byte{} {
		return 0, errors.New("empty output root. likely no proposal at timestamp")
	}
	targetTimestamp := new(big.Int).Add(output.Timestamp, finalizationPeriod)
	targetTime := time.Unix(targetTimestamp.Int64(), 0)
	// Assume clock is relatively correct
	time.Sleep(time.Until(targetTime))
	// Poll for L1 Block to have a time greater than the target time
	ticker = time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			header, err := client.HeaderByNumber(ctx, nil)
			if err != nil {
				return 0, err
			}
			if header.Time > targetTimestamp.Uint64() {
				return l2BlockNumber.Uint64(), nil
			}
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}

}

type ProofClient interface {
	GetProof(context.Context, common.Address, []string, *big.Int) (*gethclient.AccountResult, error)
}

type ReceiptClient interface {
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
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

// ProveWithdrawalParameters queries L1 & L2 to generate all withdrawal parameters and proof necessary to prove a withdrawal on L1.
// The header provided is very important. It should be a block (timestamp) for which there is a submitted output in the L2 Output Oracle
// contract. If not, the withdrawal will fail as it the storage proof cannot be verified if there is no submitted state root.
func ProveWithdrawalParameters(ctx context.Context, proofCl ProofClient, l2ReceiptCl ReceiptClient, txHash common.Hash, header *types.Header, l2OutputOracleContract *bindings.L2OutputOracleCaller) (ProvenWithdrawalParameters, error) {
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
	// Generate then verify the withdrawal proof
	withdrawalHash, err := WithdrawalHash(ev)
	if !bytes.Equal(withdrawalHash[:], ev.WithdrawalHash[:]) {
		return ProvenWithdrawalParameters{}, errors.New("Computed withdrawal hash incorrectly")
	}
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}
	slot := StorageSlotOfWithdrawalHash(withdrawalHash)
	p, err := proofCl.GetProof(ctx, predeploys.L2ToL1MessagePasserAddr, []string{slot.String()}, header.Number)
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}

	// Fetch the L2OutputIndex from the L2 Output Oracle caller (on L1)
	l2OutputIndex, err := l2OutputOracleContract.GetL2OutputIndexAfter(&bind.CallOpts{}, header.Number)
	if err != nil {
		return ProvenWithdrawalParameters{}, fmt.Errorf("failed to get l2OutputIndex: %w", err)
	}
	// TODO: Could skip this step, but it's nice to double check it
	err = VerifyProof(header.Root, p)
	if err != nil {
		return ProvenWithdrawalParameters{}, err
	}
	if len(p.StorageProof) != 1 {
		return ProvenWithdrawalParameters{}, errors.New("invalid amount of storage proofs")
	}

	// Encode it as expected by the contract
	trieNodes := make([][]byte, len(p.StorageProof[0].Proof))
	for i, s := range p.StorageProof[0].Proof {
		trieNodes[i] = common.FromHex(s)
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
			StateRoot:                header.Root,
			MessagePasserStorageRoot: p.StorageHash,
			LatestBlockhash:          header.Hash(),
		},
		WithdrawalProof: trieNodes,
	}, nil
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
	contract, err := bindings.NewL2ToL1MessagePasser(common.Address{}, nil)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		if len(log.Topics) == 0 || log.Topics[0] != MessagePassedTopic {
			continue
		}

		ev, err := contract.ParseMessagePassed(*log)
		if err != nil {
			return nil, fmt.Errorf("failed to parse log: %w", err)
		}
		return ev, nil
	}
	return nil, errors.New("Unable to find MessagePassed event")
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

package derive

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

var (
	DepositEventABI        = "TransactionDeposited(address,address,uint256,uint256,uint64,bool,bytes)"
	DepositEventABIHash    = crypto.Keccak256Hash([]byte(DepositEventABI))
	L1InfoFuncSignature    = "setL1BlockValues(uint64,uint64,uint256,bytes32,uint64)"
	L1InfoFuncBytes4       = crypto.Keccak256([]byte(L1InfoFuncSignature))[:4]
	L1InfoPredeployAddr    = common.HexToAddress("0x4200000000000000000000000000000000000015")
	L1InfoDepositerAddress = common.HexToAddress("0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001")
)

type UserDepositSource struct {
	L1BlockHash common.Hash
	LogIndex    uint64
}

const (
	UserDepositSourceDomain   = 0
	L1InfoDepositSourceDomain = 1
)

func (dep *UserDepositSource) SourceHash() common.Hash {
	var input [32 * 2]byte
	copy(input[:32], dep.L1BlockHash[:])
	binary.BigEndian.PutUint64(input[32*2-8:], dep.LogIndex)
	depositIDHash := crypto.Keccak256Hash(input[:])
	var domainInput [32 * 2]byte
	binary.BigEndian.PutUint64(domainInput[32-8:32], UserDepositSourceDomain)
	copy(domainInput[32:], depositIDHash[:])
	return crypto.Keccak256Hash(domainInput[:])
}

type L1InfoDepositSource struct {
	L1BlockHash common.Hash
	SeqNumber   uint64
}

func (dep *L1InfoDepositSource) SourceHash() common.Hash {
	var input [32 * 2]byte
	copy(input[:32], dep.L1BlockHash[:])
	binary.BigEndian.PutUint64(input[32*2-8:], dep.SeqNumber)
	depositIDHash := crypto.Keccak256Hash(input[:])

	var domainInput [32 * 2]byte
	binary.BigEndian.PutUint64(domainInput[32-8:32], L1InfoDepositSourceDomain)
	copy(domainInput[32:], depositIDHash[:])
	return crypto.Keccak256Hash(domainInput[:])
}

// UnmarshalLogEvent decodes an EVM log entry emitted by the deposit contract into typed deposit data.
//
// parse log data for:
//     event TransactionDeposited(
//    	 address indexed from,
//    	 address indexed to,
//       uint256 mint,
//    	 uint256 value,
//    	 uint64 gasLimit,
//    	 bool isCreation,
//    	 data data
//     );
//
// Additionally, the event log-index and
func UnmarshalLogEvent(ev *types.Log) (*types.DepositTx, error) {
	if len(ev.Topics) != 3 {
		return nil, fmt.Errorf("expected 3 event topics (event identity, indexed from, indexed to)")
	}
	if ev.Topics[0] != DepositEventABIHash {
		return nil, fmt.Errorf("invalid deposit event selector: %s, expected %s", ev.Topics[0], DepositEventABIHash)
	}
	if len(ev.Data) < 6*32 {
		return nil, fmt.Errorf("deposit event data too small (%d bytes): %x", len(ev.Data), ev.Data)
	}

	var dep types.DepositTx

	source := UserDepositSource{
		L1BlockHash: ev.BlockHash,
		LogIndex:    uint64(ev.Index),
	}
	dep.SourceHash = source.SourceHash()

	// indexed 0
	dep.From = common.BytesToAddress(ev.Topics[1][12:])
	// indexed 1
	to := common.BytesToAddress(ev.Topics[2][12:])

	// unindexed data
	offset := uint64(0)

	dep.Mint = new(big.Int).SetBytes(ev.Data[offset : offset+32])
	// 0 mint is represented as nil to skip minting code
	if dep.Mint.Cmp(new(big.Int)) == 0 {
		dep.Mint = nil
	}
	offset += 32

	dep.Value = new(big.Int).SetBytes(ev.Data[offset : offset+32])
	offset += 32

	gas := new(big.Int).SetBytes(ev.Data[offset : offset+32])
	if !gas.IsUint64() {
		return nil, fmt.Errorf("bad gas value: %x", ev.Data[offset:offset+32])
	}
	offset += 32
	dep.Gas = gas.Uint64()
	// isCreation: If the boolean byte is 1 then dep.To will stay nil,
	// and it will create a contract using L2 account nonce to determine the created address.
	if ev.Data[offset+31] == 0 {
		dep.To = &to
	}
	offset += 32
	// dynamic fields are encoded in three parts. The fixed size portion is the offset of the start of the
	// data. The first 32 bytes of a `bytes` object is the length of the bytes. Then are the actual bytes
	// padded out to 32 byte increments.
	var dataOffset uint256.Int
	dataOffset.SetBytes(ev.Data[offset : offset+32])
	offset += 32
	if !dataOffset.Eq(uint256.NewInt(offset)) {
		return nil, fmt.Errorf("incorrect data offset: %v", dataOffset[0])
	}

	var dataLen uint256.Int
	dataLen.SetBytes(ev.Data[offset : offset+32])
	offset += 32

	if !dataLen.IsUint64() {
		return nil, fmt.Errorf("data too large: %s", dataLen.String())
	}
	// The data may be padded to a multiple of 32 bytes
	maxExpectedLen := uint64(len(ev.Data)) - offset
	dataLenU64 := dataLen.Uint64()
	if dataLenU64 > maxExpectedLen {
		return nil, fmt.Errorf("data length too long: %d, expected max %d", dataLenU64, maxExpectedLen)
	}

	// remaining bytes fill the data
	dep.Data = ev.Data[offset : offset+dataLenU64]

	return &dep, nil
}

type L1Info interface {
	Hash() common.Hash
	ParentHash() common.Hash
	Root() common.Hash // state-root
	NumberU64() uint64
	Time() uint64
	// MixDigest field, reused for randomness after The Merge (Bellatrix hardfork)
	MixDigest() common.Hash
	BaseFee() *big.Int
	ID() eth.BlockID
	BlockRef() eth.L1BlockRef
	ReceiptHash() common.Hash
}

// L1InfoDeposit creates a L1 Info deposit transaction based on the L1 block,
// and the L2 block-height difference with the start of the epoch.
func L1InfoDeposit(seqNumber uint64, block L1Info) (*types.DepositTx, error) {
	infoDat := L1BlockInfo{
		Number:         block.NumberU64(),
		Time:           block.Time(),
		BaseFee:        block.BaseFee(),
		BlockHash:      block.Hash(),
		SequenceNumber: seqNumber,
	}
	data, err := infoDat.MarshalBinary()
	if err != nil {
		return nil, err
	}

	source := L1InfoDepositSource{
		L1BlockHash: block.Hash(),
		SeqNumber:   seqNumber,
	}
	// Uses ~30k normal case
	// Uses ~70k on first transaction
	// Round up to 75k to ensure that we always have enough gas.
	return &types.DepositTx{
		SourceHash: source.SourceHash(),
		From:       L1InfoDepositerAddress,
		To:         &L1InfoPredeployAddr,
		Mint:       nil,
		Value:      big.NewInt(0),
		Gas:        75_000,
		Data:       data,
	}, nil
}

// UserDeposits transforms the L2 block-height and L1 receipts into the transaction inputs for a full L2 block
func UserDeposits(receipts []*types.Receipt, depositContractAddr common.Address) ([]*types.DepositTx, []error) {
	var out []*types.DepositTx
	var errs []error

	for i, rec := range receipts {
		if rec.Status != types.ReceiptStatusSuccessful {
			continue
		}
		for j, log := range rec.Logs {
			if log.Address == depositContractAddr && len(log.Topics) > 0 && log.Topics[0] == DepositEventABIHash {
				dep, err := UnmarshalLogEvent(log)
				if err != nil {
					errs = append(errs, fmt.Errorf("malformatted L1 deposit log in receipt %d, log %d: %w", i, j, err))
				} else {
					out = append(out, dep)
				}
			}
		}
	}
	return out, errs
}

func BatchesFromEVMTransactions(config *rollup.Config, txLists []types.Transactions) ([]*BatchData, error) {
	var out []*BatchData
	l1Signer := config.L1Signer()
	for _, txs := range txLists {
		for _, tx := range txs {
			if to := tx.To(); to != nil && *to == config.BatchInboxAddress {
				seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
				if err != nil {
					// TODO: log error
					continue // bad signature, ignore
				}
				// some random L1 user might have sent a transaction to our batch inbox, ignore them
				if seqDataSubmitter != config.BatchSenderAddress {
					// TODO: log/record metric
					continue // not an authorized batch submitter, ignore
				}
				batches, err := DecodeBatches(config, bytes.NewReader(tx.Data()))
				if err != nil {
					// TODO: log/record metric
					continue
				}
				out = append(out, batches...)
			}
		}
	}
	return out, nil
}

func FilterBatches(config *rollup.Config, epoch rollup.Epoch, minL2Time uint64, maxL2Time uint64, batches []*BatchData) (out []*BatchData) {
	uniqueTime := make(map[uint64]struct{})
	for _, batch := range batches {
		if !ValidBatch(batch, config, epoch, minL2Time, maxL2Time) {
			continue
		}
		// Check if we have already seen a batch for this L2 block
		if _, ok := uniqueTime[batch.Timestamp]; ok {
			// block already exists, batch is duplicate (first batch persists, others are ignored)
			continue
		}
		uniqueTime[batch.Timestamp] = struct{}{}
		out = append(out, batch)
	}
	return
}

func ValidBatch(batch *BatchData, config *rollup.Config, epoch rollup.Epoch, minL2Time uint64, maxL2Time uint64) bool {
	if batch.Epoch != epoch {
		// Batch was tagged for past or future epoch,
		// i.e. it was included too late or depends on the given L1 block to be processed first.
		return false
	}
	if (batch.Timestamp-config.Genesis.L2Time)%config.BlockTime != 0 {
		return false // bad timestamp, not a multiple of the block time
	}
	if batch.Timestamp < minL2Time {
		return false // old batch
	}
	// limit timestamp upper bound to avoid huge amount of empty blocks
	if batch.Timestamp >= maxL2Time {
		return false // too far in future
	}
	for _, txBytes := range batch.Transactions {
		if len(txBytes) == 0 {
			return false // transaction data must not be empty
		}
		if txBytes[0] == types.DepositTxType {
			return false // sequencers may not embed any deposits into batch data
		}
	}
	return true
}

type L2Info interface {
	Time() uint64
}

// FillMissingBatches turns a collection of batches to the input batches for a series of blocks
func FillMissingBatches(batches []*BatchData, epoch, blockTime, minL2Time, nextL1Time uint64) []*BatchData {
	m := make(map[uint64]*BatchData)
	// The number of L2 blocks per sequencing window is variable, we do not immediately fill to maxL2Time:
	// - ensure at least 1 block
	// - fill up to the next L1 block timestamp, if higher, to keep up with L1 time
	// - fill up to the last valid batch, to keep up with L2 time
	newHeadL2Timestamp := minL2Time
	if nextL1Time > newHeadL2Timestamp+blockTime {
		newHeadL2Timestamp = nextL1Time - blockTime
	}
	for _, b := range batches {
		m[b.BatchV1.Timestamp] = b
		if b.Timestamp > newHeadL2Timestamp {
			newHeadL2Timestamp = b.Timestamp
		}
	}
	var out []*BatchData
	for t := minL2Time; t <= newHeadL2Timestamp; t += blockTime {
		b, ok := m[t]
		if ok {
			out = append(out, b)
		} else {
			out = append(out, &BatchData{
				BatchV1{
					Epoch:     rollup.Epoch(epoch),
					Timestamp: t,
				},
			})
		}
	}
	return out
}

// L1InfoDepositBytes returns a serialized L1-info attributes transaction.
func L1InfoDepositBytes(seqNumber uint64, l1Info L1Info) (hexutil.Bytes, error) {
	dep, err := L1InfoDeposit(seqNumber, l1Info)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 info tx: %v", err)
	}
	l1Tx := types.NewTx(dep)
	opaqueL1Tx, err := l1Tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to encode L1 info tx: %v", err)
	}
	return opaqueL1Tx, nil
}

func DeriveDeposits(receipts []*types.Receipt, depositContractAddr common.Address) ([]hexutil.Bytes, []error) {
	userDeposits, errs := UserDeposits(receipts, depositContractAddr)
	encodedTxs := make([]hexutil.Bytes, 0, len(userDeposits))
	for i, tx := range userDeposits {
		opaqueTx, err := types.NewTx(tx).MarshalBinary()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to encode user tx %d", i))
		} else {
			encodedTxs = append(encodedTxs, opaqueTx)
		}
	}
	return encodedTxs, errs
}

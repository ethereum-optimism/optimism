package types

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ContainsQuery struct {
	Timestamp uint64
	BlockNum  uint64
	LogIdx    uint32
	Checksum  MessageChecksum
}

type ExecutingMessage struct {
	ChainID   eth.ChainID
	BlockNum  uint64
	LogIdx    uint32
	Timestamp uint64
	Checksum  MessageChecksum
}

func (s *ExecutingMessage) String() string {
	return fmt.Sprintf("ExecMsg(chain: %s, block: %d, log: %d, time: %d, checksum: %s)",
		s.ChainID, s.BlockNum, s.LogIdx, s.Timestamp, s.Checksum)
}

type MessageChecksum common.Hash

func (mc MessageChecksum) MarshalText() ([]byte, error) {
	return common.Hash(mc).MarshalText()
}

func (mc *MessageChecksum) UnmarshalText(data []byte) error {
	return (*common.Hash)(mc).UnmarshalText(data)
}

func (mc MessageChecksum) String() string {
	return common.Hash(mc).String()
}

type ChecksumArgs struct {
	BlockNumber uint64
	LogIndex    uint32
	Timestamp   uint64
	ChainID     eth.ChainID
	LogHash     common.Hash
}

func (args ChecksumArgs) Checksum() MessageChecksum {
	idPacked := make([]byte, 12, 32) // 12 zero bytes, as padding to 32 bytes
	idPacked = binary.BigEndian.AppendUint64(idPacked, args.BlockNumber)
	idPacked = binary.BigEndian.AppendUint64(idPacked, args.Timestamp)
	idPacked = binary.BigEndian.AppendUint32(idPacked, args.LogIndex)
	idLogHash := crypto.Keccak256Hash(args.LogHash[:], idPacked)
	chainID := args.ChainID.Bytes32()
	out := crypto.Keccak256Hash(idLogHash[:], chainID[:])
	out[0] = 0x03 // type/version byte
	return MessageChecksum(out)
}

type BlockSeal struct {
	Hash      common.Hash `json:"hash"`
	Number    uint64      `json:"number"`
	Timestamp uint64      `json:"timestamp"`
}

func (s BlockSeal) String() string {
	return fmt.Sprintf("BlockSeal(hash:%s, number:%d, time:%d)", s.Hash, s.Number, s.Timestamp)
}

func (s BlockSeal) ID() eth.BlockID {
	return eth.BlockID{Hash: s.Hash, Number: s.Number}
}

// Marshaling helpers for any JSON encoded usage of these types

type blockSealMarshaling struct {
	Hash      common.Hash  `json:"hash"`
	Number    hexutil.Uint64 `json:"number"`
	Timestamp hexutil.Uint64 `json:"timestamp"`
}

func (s BlockSeal) MarshalJSON() ([]byte, error) {
	enc := blockSealMarshaling{
		Hash:      s.Hash,
		Number:    hexutil.Uint64(s.Number),
		Timestamp: hexutil.Uint64(s.Timestamp),
	}
	return json.Marshal(&enc)
}

func (s *BlockSeal) UnmarshalJSON(input []byte) error {
	var dec blockSealMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	s.Hash = dec.Hash
	s.Number = uint64(dec.Number)
	s.Timestamp = uint64(dec.Timestamp)
	return nil
}

// Simple validation helpers mirrored from supervisor/types for compatibility where needed

var (
	errLogIndexTooLarge = errors.New("log index too large")
)

type Identifier struct {
	Origin      common.Address
	BlockNumber uint64
	LogIndex    uint32
	Timestamp   uint64
	ChainID     eth.ChainID
}

type identifierMarshaling struct {
	Origin      common.Address `json:"origin"`
	BlockNumber hexutil.Uint64 `json:"blockNumber"`
	LogIndex    hexutil.Uint64 `json:"logIndex"`
	Timestamp   hexutil.Uint64 `json:"timestamp"`
	ChainID     hexutil.U256   `json:"chainID"`
}

func (id Identifier) MarshalJSON() ([]byte, error) {
	var enc identifierMarshaling
	enc.Origin = id.Origin
	enc.BlockNumber = hexutil.Uint64(id.BlockNumber)
	enc.LogIndex = hexutil.Uint64(id.LogIndex)
	enc.Timestamp = hexutil.Uint64(id.Timestamp)
	enc.ChainID = (hexutil.U256)(id.ChainID)
	return json.Marshal(&enc)
}

func (id *Identifier) UnmarshalJSON(input []byte) error {
	var dec identifierMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	id.Origin = dec.Origin
	id.BlockNumber = uint64(dec.BlockNumber)
	if dec.LogIndex > math.MaxUint32 {
		return fmt.Errorf("%w: %d", errLogIndexTooLarge, dec.LogIndex)
	}
	id.LogIndex = uint32(dec.LogIndex)
	id.Timestamp = uint64(dec.Timestamp)
	id.ChainID = (eth.ChainID)(dec.ChainID)
	return nil
}



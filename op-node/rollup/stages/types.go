package stages

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
)

const (
	FrameVersion0 byte = 0
	// FrameVersion0MaxHeaderSize = version (byte) + offset (uvarint) + end block hash (bytes32) + delta num (uvarint)
	FrameVersion0MaxHeaderSize = 1 + binary.MaxVarintLen64 + 32 + binary.MaxVarintLen64

	ChunkVersion0 byte = 0
	// ChunkVersion0MaxHeaderSize = version (byte) + chunk num (uvarint) + start block hash (bytes32) + start num (uvarint)
	ChunkVersion0MaxHeaderSize = 1 + binary.MaxVarintLen64 + 32 + binary.MaxVarintLen64

	// TODO: I just picked snappy compression as placeholder since we already use it on gossip-sub and I'm familiar with the API.
	// But we should be using the compression algo that was picked for mainnet previously.
	CompressionVersion0             = 0
	CompressionVersion0MaxFrameSize = 20_000
)

type FrameHeader struct {
	Version byte
	// Offset to the start of the Content
	Offset  uint64
	End     eth.BlockID
	Content []byte
}

func (frame *FrameHeader) HeaderSize() uint64 {
	var tmp [binary.MaxVarintLen64]byte
	n := 1
	n += binary.PutUvarint(tmp[:], frame.Offset)
	n += 32
	n += binary.PutUvarint(tmp[:], frame.End.Number)
	return uint64(n)
}

func (frame *FrameHeader) MarshalBinary() ([]byte, error) {
	out := make([]byte, FrameVersion0MaxHeaderSize+len(frame.Content))
	out[0] = FrameVersion0
	n := 1
	n += binary.PutUvarint(out[n:], frame.Offset)
	n += copy(out[n:], frame.End.Hash[:])
	n += binary.PutUvarint(out[n:], frame.End.Number)
	n += copy(out[n:], frame.Content)
	out = out[:n]
	return out, nil
}

func (frame *FrameHeader) UnmarshalBinary(data []byte) error {
	r := bytes.NewReader(data)
	var err error
	frame.Version, err = r.ReadByte()
	if err != nil {
		return fmt.Errorf("cannot read frame version: %v", err)
	}
	if frame.Version != FrameVersion0 {
		return fmt.Errorf("unexpected frame version: %d", frame.Version)
	}
	frame.Offset, err = binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("cannot read frame offset: %v", err)
	}
	frame.End.Hash, err = readHash(r)
	if err != nil {
		return fmt.Errorf("cannot read frame end block hash: %v", err)
	}
	frame.End.Number, err = binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("cannot read frame end block number: %v", err)
	}
	return nil
}

type ChunkHeader struct {
	Version  byte
	ChunkNum uint64
	Start    eth.BlockID
}

func uvarint(x uint64) []byte {
	var tmp [binary.MaxVarintLen64]byte
	return tmp[:binary.PutUvarint(tmp[:], x)]
}

func readHash(r io.Reader) (out common.Hash, err error) {
	_, err = io.ReadFull(r, out[:])
	return
}

func (chunk *ChunkHeader) HeaderSize() uint64 {
	var tmp [binary.MaxVarintLen64]byte
	n := 1
	n += binary.PutUvarint(tmp[:], chunk.ChunkNum)
	n += 32
	n += binary.PutUvarint(tmp[:], chunk.Start.Number)
	return uint64(n)
}

func (chunk *ChunkHeader) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(ChunkVersion0)
	buf.Write(uvarint(chunk.ChunkNum))
	buf.Write(chunk.Start.Hash[:])
	buf.Write(uvarint(chunk.Start.Number))
	return buf.Bytes(), nil
}

func (chunk *ChunkHeader) UnmarshalBinary(data []byte) error {
	r := bytes.NewReader(data)
	var err error
	chunk.Version, err = r.ReadByte()
	if err != nil {
		return fmt.Errorf("cannot read chunk version: %v", err)
	}
	if chunk.Version != ChunkVersion0 {
		return fmt.Errorf("unexpected chunk version: %d", chunk.Version)
	}
	chunk.ChunkNum, err = binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("cannot read chunk number: %v", err)
	}
	chunk.Start.Hash, err = readHash(r)
	if err != nil {
		return fmt.Errorf("cannot read chunk start block hash: %v", err)
	}
	chunk.Start.Number, err = binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("cannot read chunk start block number: %v", err)
	}
	return nil
}

type Frame struct {
	FrameHeader
	Content []byte
}

type Chunk struct {
	ChunkHeader
	Frames []Frame
}

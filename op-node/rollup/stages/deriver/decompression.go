package deriver

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/rollup/stages"
	"github.com/golang/snappy"
	"io"
	"sync"
)

type byteReader struct {
	io.Reader
}

func (b *byteReader) ReadByte() (byte, error) {
	var tmp [1]byte
	_, err := b.Read(tmp[:])
	return tmp[0], err
}

type Decompressor struct {
	mu    sync.Mutex
	Inner BinaryReaderStage
	buf   []byte
}

func (cs *Decompressor) next() error {
	br := byteReader{cs.Inner}
	version, err := br.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read compression version: %w", err)
	}
	if version != stages.CompressionVersion0 {
		return fmt.Errorf("unknown compression version: %d", version)
	}

	compressedLen, err := binary.ReadUvarint(&br)
	if err != nil {
		return fmt.Errorf("failed to read compressed length: %w", err)
	}
	if compressedLen > uint64(snappy.MaxEncodedLen(stages.CompressionVersion0MaxFrameSize)) {
		return fmt.Errorf("read compressed length is too large: %d", compressedLen)
	}
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, cs.Inner, int64(compressedLen)); err != nil {
		return fmt.Errorf("failed to fully read compression frame: %v", err)
	}
	inData := buf.Bytes()
	// zip-bomb protection
	if len(inData) > stages.CompressionVersion0MaxFrameSize {
		return fmt.Errorf("bad compressed data, length %d is too large", len(inData))
	}
	out, err := snappy.Decode(nil, inData)
	if err != nil {
		return fmt.Errorf("failed to decompress: %v", err)
	}
	cs.buf = out
	return nil
}

func (cs *Decompressor) Read(p []byte) (n int, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if len(cs.buf) == 0 {
		if err := cs.next(); err != nil {
			return 0, err
		}
	}
	n = copy(p, cs.buf)
	cs.buf = cs.buf[n:]
	return n, nil
}

func (cs *Decompressor) Close() error {
	return cs.Inner.Close()
}

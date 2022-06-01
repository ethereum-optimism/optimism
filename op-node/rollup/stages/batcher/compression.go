package batcher

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/rollup/stages"
	"io"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/golang/snappy"
)

type Compressor struct {
	mu sync.Mutex

	Inner BinaryReaderStage
	Buf   []byte
}

func (cs *Compressor) Reset(start, end eth.BlockID) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Reset(start, end)
}

func (cs *Compressor) nextCompressFrame() error {
	// TODO pool this
	var buf bytes.Buffer
	n, err := io.CopyN(&buf, cs.Inner, stages.CompressionVersion0MaxFrameSize)
	if err == io.EOF {
		if n == 0 {
			// No more contents left to compress
			return io.EOF
		} else {
			// we are compressing less than CompressionVersion0MaxFrameSize
			err = nil
		}
	}
	if err != nil {
		return fmt.Errorf("failed to read input data for next compression frame: %w", err)
	}

	// TODO pool this
	dest := make([]byte, 1+snappy.MaxEncodedLen(stages.CompressionVersion0MaxFrameSize))
	dest[0] = stages.CompressionVersion0
	x := binary.PutUvarint(dest[1:], uint64(len(buf.Bytes())))
	dest = snappy.Encode(dest[1+x:], buf.Bytes())
	cs.Buf = dest
	return nil
}

func (cs *Compressor) Read(dest []byte) (n int, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for {
		if len(dest) == 0 {
			return n, nil
		}
		if len(cs.Buf) > 0 {
			copied := copy(dest, cs.Buf)
			dest = dest[copied:]
			cs.Buf = cs.Buf[copied:]
			n += copied
			continue
		}
		if len(cs.Buf) == 0 {
			err := cs.nextCompressFrame()
			if err != nil {
				return n, err
			}
		}
	}
}

func (cs *Compressor) Start() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Start()
}

func (cs *Compressor) End() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.End()
}

func (cs *Compressor) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	// TODO: release data back into pool
	cs.Buf = nil
	return cs.Inner.Close()
}

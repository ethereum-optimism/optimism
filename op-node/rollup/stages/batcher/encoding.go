package batcher

import (
	"fmt"
	"io"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	EncoderVersion0 = 0
)

type Encoder struct {
	mu sync.Mutex

	Inner         BatchReaderStage
	CurrentReader io.Reader
}

func (cs *Encoder) Reset(start, end eth.BlockID) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Reset(start, end)
}

func (cs *Encoder) nextBatch() error {
	var batch derive.BatchData
	err := cs.Inner.Read(&batch)
	if err != nil {
		if err == io.EOF { // when there are no more inputs
			return io.EOF
		} else {
			return fmt.Errorf("failed to read next batch: %w", err)
		}
	}
	_, r, err := rlp.EncodeToReader(batch)
	if err != nil {
		return fmt.Errorf("failed to create RLP encoder: %w", err)
	}
	cs.CurrentReader = r
	return nil
}

func (cs *Encoder) Read(dest []byte) (n int, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for {
		// stop if we can't read more data
		if len(dest) == 0 {
			return n, nil
		}
		if cs.CurrentReader == nil {
			if err := cs.nextBatch(); err != nil {
				return n, err
			}
		}
		innerN, err := cs.CurrentReader.Read(dest)
		// remember how much we've read
		n += innerN
		dest = dest[innerN:]
		if err == io.EOF {
			cs.CurrentReader = nil
			continue
		}
		if err != nil {
			return n, err
		}
	}
}

func (cs *Encoder) Start() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Start()
}

func (cs *Encoder) End() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.End()
}

func (cs *Encoder) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.CurrentReader != nil {
		// we need to read to EOF to release the RLP reader back into the pool (it has no Close())
		_, _ = io.Copy(io.Discard, cs.CurrentReader)
		cs.CurrentReader = nil
	}
	return cs.Inner.Close()
}

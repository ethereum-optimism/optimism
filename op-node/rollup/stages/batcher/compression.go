package batcher

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/eth"
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

func (cs *Compressor) Read(dest []byte) (n int, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// TODO: cw, err := zlib.NewWriterLevelDict(w, zlib.BestCompression, nil)
	return 0, nil
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

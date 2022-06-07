package deriver

import (
	"compress/zlib"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/rollup/stages"
	"io"
	"sync"
)

type Decompressor struct {
	mu         sync.Mutex
	openReader io.ReadCloser
	Inner      BinaryReaderStage
}

func (cs *Decompressor) Read(p []byte) (n int, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if len(p) == 0 {
		return 0, nil
	}

	if cs.openReader == nil {
		var versionByte [1]byte
		if _, err := io.ReadFull(cs.Inner, versionByte[:]); err != nil {
			return 0, fmt.Errorf("failed to read version byte: %v", err)
		}
		switch versionByte[0] {
		case stages.CompressionVersion0:
			r, err := zlib.NewReaderDict(cs.Inner, nil) // TODO: maybe provide a dict for better performance.
			if err != nil {
				return 0, fmt.Errorf("failed to create new zlib reader: %v", err)
			}
			cs.openReader = r
		// new compression types or configurations, may be supported in the future.
		default:
			return 0, fmt.Errorf("unknown compression version: %d", versionByte[0])
		}
	}

	// We assume here that the reader internally, for temporary allocations,
	// is not vulnerable to zip-bombs etc. We only read as much from the reader as is necessary.
	return cs.openReader.Read(p)
}

func (cs *Decompressor) Close() error {
	return cs.Inner.Close()
}

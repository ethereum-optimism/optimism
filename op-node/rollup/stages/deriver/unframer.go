package deriver

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/rollup/stages"
	"io"
	"sync"
)

type Unframer struct {
	mu     sync.Mutex
	Inner  FrameReaderStage
	buf    []byte
	offset uint64
}

func (cs *Unframer) Read(dest []byte) (n int, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	var frame stages.Frame
	if err := cs.Inner.Read(&frame); err == io.EOF {
		return 0, io.EOF
	} else if err != nil {
		return 0, fmt.Errorf("failed to pull next frame: %w", err)
	}
	if frame.Offset != cs.offset {
		return 0, fmt.Errorf("frame starts at wrong offset: %d, expected %d", frame.Offset, cs.offset)
	}
	cs.offset += uint64(len(frame.Content))
	cs.buf = append(cs.buf, frame.Content...)
	n = copy(dest, cs.buf)
	// TODO: recycle buf better
	cs.buf = cs.buf[n:]
	return n, nil
}

func (cs *Unframer) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Close()
}

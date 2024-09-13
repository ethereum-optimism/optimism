package exec

import (
	"encoding/binary"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
)

type PreimageReader interface {
	ReadPreimage(key [32]byte, offset uint32) (dat [32]byte, datLen uint32)
}

// TrackingPreimageOracleReader wraps around a PreimageOracle, implements the PreimageOracle interface, and adds tracking functionality.
// It also implements the PreimageReader interface
type TrackingPreimageOracleReader struct {
	po mipsevm.PreimageOracle

	totalPreimageSize   int
	numPreimageRequests int

	// cached pre-image data, including 8 byte length prefix
	lastPreimage []byte
	// key for above preimage
	lastPreimageKey [32]byte
	// offset we last read from, or max uint32 if nothing is read this step
	lastPreimageOffset uint32
}

func NewTrackingPreimageOracleReader(po mipsevm.PreimageOracle) *TrackingPreimageOracleReader {
	return &TrackingPreimageOracleReader{po: po}
}

func (p *TrackingPreimageOracleReader) Reset() {
	p.lastPreimageOffset = ^uint32(0)
}

func (p *TrackingPreimageOracleReader) Hint(v []byte) {
	p.po.Hint(v)
}

func (p *TrackingPreimageOracleReader) GetPreimage(k [32]byte) []byte {
	p.numPreimageRequests++
	preimage := p.po.GetPreimage(k)
	p.totalPreimageSize += len(preimage)
	return preimage
}

func (p *TrackingPreimageOracleReader) ReadPreimage(key [32]byte, offset uint32) (dat [32]byte, datLen uint32) {
	preimage := p.lastPreimage
	if key != p.lastPreimageKey {
		p.lastPreimageKey = key
		data := p.GetPreimage(key)
		// add the length prefix
		preimage = make([]byte, 0, 8+len(data))
		preimage = binary.BigEndian.AppendUint64(preimage, uint64(len(data)))
		preimage = append(preimage, data...)
		p.lastPreimage = preimage
	}
	p.lastPreimageOffset = offset
	if offset >= uint32(len(preimage)) {
		panic("Preimage offset out-of-bounds")
	}
	datLen = uint32(copy(dat[:], preimage[offset:]))
	return
}

func (p *TrackingPreimageOracleReader) LastPreimage() ([32]byte, []byte, uint32) {
	return p.lastPreimageKey, p.lastPreimage, p.lastPreimageOffset
}

func (p *TrackingPreimageOracleReader) TotalPreimageSize() int {
	return p.totalPreimageSize
}

func (p *TrackingPreimageOracleReader) NumPreimageRequests() int {
	return p.numPreimageRequests
}

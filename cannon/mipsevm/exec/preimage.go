package exec

import (
	"encoding/binary"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
)

// TrackingOracle wraps around a PreimageOracle, implements the PreimageOracle interface, and adds tracking functionality
type TrackingOracle struct {
	po                  mipsevm.PreimageOracle
	TotalPreimageSize   int
	NumPreimageRequests int
}

func NewTrackingOracle(po mipsevm.PreimageOracle) *TrackingOracle {
	return &TrackingOracle{po: po}
}

func (d *TrackingOracle) Hint(v []byte) {
	d.po.Hint(v)
}

func (d *TrackingOracle) GetPreimage(k [32]byte) []byte {
	d.NumPreimageRequests++
	preimage := d.po.GetPreimage(k)
	d.TotalPreimageSize += len(preimage)
	return preimage
}

type PreimageReader struct {
	po mipsevm.PreimageOracle

	// cached pre-image data, including 8 byte length prefix
	lastPreimage []byte
	// key for above preimage
	lastPreimageKey [32]byte
	// offset we last read from, or max uint32 if nothing is read this step
	lastPreimageOffset uint32
}

func NewPreimageReader(po mipsevm.PreimageOracle) *PreimageReader {
	return &PreimageReader{po: po}
}

func (p *PreimageReader) Reset() {
	p.lastPreimageOffset = ^uint32(0)
}

func (p *PreimageReader) readPreimage(key [32]byte, offset uint32) (dat [32]byte, datLen uint32) {
	preimage := p.lastPreimage
	if key != p.lastPreimageKey {
		p.lastPreimageKey = key
		data := p.po.GetPreimage(key)
		// add the length prefix
		preimage = make([]byte, 0, 8+len(data))
		preimage = binary.BigEndian.AppendUint64(preimage, uint64(len(data)))
		preimage = append(preimage, data...)
		p.lastPreimage = preimage
	}
	p.lastPreimageOffset = offset
	datLen = uint32(copy(dat[:], preimage[offset:]))
	return
}

func (p *PreimageReader) LastPreimage() ([32]byte, []byte, uint32) {
	return p.lastPreimageKey, p.lastPreimage, p.lastPreimageOffset
}

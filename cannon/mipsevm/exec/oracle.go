package exec

import "github.com/ethereum-optimism/optimism/cannon/mipsevm"

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

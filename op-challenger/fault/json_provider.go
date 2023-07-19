package fault

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// JsonProvider is a [TraceProvider] that provides claims for specific
// indices in the given trace.
type JsonProvider struct {
	files   []uint64
	maxSize uint64
}

// NewJsonProvider returns a new [JsonProvider].
// It accepts a `traceDir` which is the directory containing json failes whose names
// are the trace indicies. It also accepts the `depth` of the trace.
func NewJsonProvider(traceDir string, depth uint64) (*JsonProvider, error) {
	// Read the files in the trace directory.
	files, err := ioutil.ReadDir(traceDir)
	if err != nil {
		return nil, err
	}
	names := make([]uint64, len(files))
	for i, file := range files {
		u, err := strconv.ParseUint(file.Name()[:strings.IndexByte(file.Name(), '.')], 0, 64)
		if err != nil {
			return nil, err
		}
		names[i] = u
	}
	return &JsonProvider{
		files:   names,
		maxSize: uint64(1 << depth),
	}, nil
}

// GetPreimage returns the preimage for the given hash.
func (j *JsonProvider) GetPreimage(i uint64) ([]byte, error) {
	// The index cannot be larger than the maximum index as computed by the depth.
	if i >= j.maxSize {
		return []byte{}, ErrIndexTooLarge
	}
	// We extend the deepest hash to the maximum depth if the trace is not expansive.
	if i >= uint64(len(j.files)) {
		return j.GetPreimage(uint64(len(j.files)) - 1)
	}
	return IndexToBytes(j.files[i]), nil
}

// Get returns the claim value at the given index in the trace.
func (j *JsonProvider) Get(i uint64) (common.Hash, error) {
	claimBytes, err := j.GetPreimage(i)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(claimBytes), nil
}

// AbsolutePreState returns the preimage of the entire dispute game.
// This is the undisputed starting state of the game.
func (j *JsonProvider) AbsolutePreState() []byte {
	out := make([]byte, 32)
	out[31] = 140
	return out
}

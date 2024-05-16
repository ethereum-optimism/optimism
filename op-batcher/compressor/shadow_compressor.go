package compressor

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

const (
	// safeCompressionOverhead is the largest potential blow-up in bytes we expect to see when
	// compressing arbitrary (e.g. random) data.  Here we account for a 2 byte header, 4 byte
	// digest, 5 byte EOF indicator, and then 5 byte flate block header for each 16k of potential
	// data. Assuming frames are max 128k size (the current max blob size) this is 2+4+5+(5*8) = 51
	// bytes.  If we start using larger frames (e.g. should max blob size increase) a larger blowup
	// might be possible, but it would be highly unlikely, and the system still works if our
	// estimate is wrong -- we just end up writing one more tx for the overflow.
	safeCompressionOverhead = 51
)

type ShadowCompressor struct {
	config Config

	compressor       derive.ChannelCompressor
	shadowCompressor derive.ChannelCompressor

	fullErr error

	bound uint64 // best known upperbound on the size of the compressed output
}

// NewShadowCompressor creates a new derive.Compressor implementation that contains two
// compression buffers: one used for size estimation, and one used for the final
// compressed output. The first is flushed on every write, the second isn't, which means
// the final compressed data is always slightly smaller than the target. There is one
// exception to this rule: the first write to the buffer is not checked against the
// target, which allows individual blocks larger than the target to be included (and will
// be split across multiple channel frames).
func NewShadowCompressor(config Config) (derive.Compressor, error) {
	c := &ShadowCompressor{
		config: config,
	}

	var err error
	c.compressor, err = derive.NewChannelCompressor(config.CompressionAlgo)
	if err != nil {
		return nil, err
	}
	c.shadowCompressor, err = derive.NewChannelCompressor(config.CompressionAlgo)
	if err != nil {
		return nil, err
	}

	c.bound = safeCompressionOverhead
	return c, nil
}

func (t *ShadowCompressor) Write(p []byte) (int, error) {
	if t.fullErr != nil {
		return 0, t.fullErr
	}
	_, err := t.shadowCompressor.Write(p)
	if err != nil {
		return 0, err
	}
	newBound := t.bound + uint64(len(p))
	if newBound > t.config.TargetOutputSize {
		// Do not flush the buffer unless there's some chance we will be over the size limit.
		// This reduces CPU but more importantly it makes the shadow compression ratio more
		// closely reflect the ultimate compression ratio.
		if err = t.shadowCompressor.Flush(); err != nil {
			return 0, err
		}
		newBound = uint64(t.shadowCompressor.Len()) + CloseOverheadZlib
		if newBound > t.config.TargetOutputSize {
			t.fullErr = derive.ErrCompressorFull
			if t.Len() > 0 {
				// only return an error if we've already written data to this compressor before
				// (otherwise single blocks over the target would never be written)
				return 0, t.fullErr
			}
		}
	}
	t.bound = newBound
	return t.compressor.Write(p)
}

func (t *ShadowCompressor) Close() error {
	return t.compressor.Close()
}

func (t *ShadowCompressor) Read(p []byte) (int, error) {
	return t.compressor.Read(p)
}

func (t *ShadowCompressor) Reset() {
	t.compressor.Reset()
	t.shadowCompressor.Reset()
	t.fullErr = nil
	t.bound = safeCompressionOverhead
}

func (t *ShadowCompressor) Len() int {
	return t.compressor.Len()
}

func (t *ShadowCompressor) Flush() error {
	return t.compressor.Flush()
}

func (t *ShadowCompressor) FullErr() error {
	return t.fullErr
}

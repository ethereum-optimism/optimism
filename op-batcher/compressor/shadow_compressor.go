package compressor

import (
	"bytes"
	"compress/zlib"

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

	buf      bytes.Buffer
	compress *zlib.Writer

	shadowBuf      bytes.Buffer
	shadowCompress *zlib.Writer

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
	c.compress, err = zlib.NewWriterLevel(&c.buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	c.shadowCompress, err = zlib.NewWriterLevel(&c.shadowBuf, zlib.BestCompression)
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
	_, err := t.shadowCompress.Write(p)
	if err != nil {
		return 0, err
	}
	newBound := t.bound + uint64(len(p))
	cap := t.config.TargetFrameSize * uint64(t.config.TargetNumFrames)
	if newBound > cap {
		// Do not flush the buffer unless there's some chance we will be over the size limit.
		// This reduces CPU but more importantly it makes the shadow compression ratio more
		// closely reflect the ultimate compression ratio.
		err = t.shadowCompress.Flush()
		if err != nil {
			return 0, err
		}
		newBound = uint64(t.shadowBuf.Len()) + 4 // + 4 is to account for the digest written on close()
		if newBound > cap {
			t.fullErr = derive.CompressorFullErr
			if t.Len() > 0 {
				// only return an error if we've already written data to this compressor before
				// (otherwise individual blocks over the target would never be written)
				return 0, t.fullErr
			}
		}
	}
	t.bound = newBound
	return t.compress.Write(p)
}

func (t *ShadowCompressor) Close() error {
	return t.compress.Close()
}

func (t *ShadowCompressor) Read(p []byte) (int, error) {
	return t.buf.Read(p)
}

func (t *ShadowCompressor) Reset() {
	t.buf.Reset()
	t.compress.Reset(&t.buf)
	t.shadowBuf.Reset()
	t.shadowCompress.Reset(&t.shadowBuf)
	t.fullErr = nil
	t.bound = safeCompressionOverhead
}

func (t *ShadowCompressor) Len() int {
	return t.buf.Len()
}

func (t *ShadowCompressor) Flush() error {
	return t.compress.Flush()
}

func (t *ShadowCompressor) FullErr() error {
	return t.fullErr
}

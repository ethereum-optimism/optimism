package compressor

import (
	"bytes"
	"compress/zlib"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type ShadowCompressor struct {
	config Config

	buf      bytes.Buffer
	compress *zlib.Writer

	shadowBuf      bytes.Buffer
	shadowCompress *zlib.Writer

	fullErr error
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

	return c, nil
}

func (t *ShadowCompressor) Write(p []byte) (int, error) {
	_, err := t.shadowCompress.Write(p)
	if err != nil {
		return 0, err
	}
	err = t.shadowCompress.Flush()
	if err != nil {
		return 0, err
	}
	if uint64(t.shadowBuf.Len()) > t.config.TargetFrameSize*uint64(t.config.TargetNumFrames) {
		t.fullErr = derive.CompressorFullErr
		if t.Len() > 0 {
			// only return an error if we've already written data to this compressor before
			// (otherwise individual blocks over the target would never be written)
			return 0, t.fullErr
		}
	}
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

package batcher

import (
	"bytes"
	"compress/zlib"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type ShadowCompressor struct {
	// The frame size to target when creating channel frames. When adding new
	// data to the shadow compressor causes the buffer size to be greater than
	// the target size, the compressor is marked as full.
	TargetFrameSize uint64
	// The target number of frames to create in this channel. If the realized
	// compression ratio is worse than approxComprRatio, additional leftover
	// frame(s) might get created.
	TargetNumFrames int

	buf      bytes.Buffer
	compress *zlib.Writer

	shadowBuf      bytes.Buffer
	shadowCompress *zlib.Writer

	fullErr error
}

func NewShadowCompressor(targetFrameSize uint64, targetNumFrames int) (derive.Compressor, error) {
	c := &ShadowCompressor{
		TargetFrameSize: targetFrameSize,
		TargetNumFrames: targetNumFrames,
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
	if t.Len() > 0 {
		err = t.shadowCompress.Flush()
		if err != nil {
			return 0, err
		}
		if uint64(t.shadowBuf.Len()) > t.TargetFrameSize*uint64(t.TargetNumFrames) {
			t.fullErr = derive.CompressorFullErr
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

package compressor

import (
	"bytes"
	"compress/zlib"
	"fmt"

	"github.com/DataDog/zstd"
	"github.com/andybalholm/brotli"

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

	zlibBuf      bytes.Buffer
	brotliBuf   bytes.Buffer
	zstdBuf   bytes.Buffer
	compress *zlib.Writer
	brotliCompress *brotli.Writer
	zstdCompress *zstd.Writer

	shadowBuf      bytes.Buffer
	shadowCompress *zlib.Writer

	compressAlgo string

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
	fmt.Println("NewShadowCompressor")
	c := &ShadowCompressor{
		config: config,
	}

	var err error
	c.compress, err = zlib.NewWriterLevel(&c.zlibBuf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	c.shadowCompress, err = zlib.NewWriterLevel(&c.shadowBuf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}

	// add brotli
	c.brotliCompress = brotli.NewWriterLevel(
		&c.brotliBuf,
		brotli.BestCompression,
	)

	// add zstd
	c.zstdCompress = zstd.NewWriterLevel(&c.zstdBuf, 22)

	c.compressAlgo = config.CompressionAlgo

	c.bound = safeCompressionOverhead
	return c, nil
}

func (t *ShadowCompressor) Write(p []byte) (int, error) {
	fmt.Println(t.compressAlgo)
	if t.fullErr != nil {
		return 0, t.fullErr
	}
	_, err := t.shadowCompress.Write(p)
	if err != nil {
		return 0, err
	}
	newBound := t.bound + uint64(len(p))
	if newBound > t.config.TargetOutputSize {
		// Do not flush the buffer unless there's some chance we will be over the size limit.
		// This reduces CPU but more importantly it makes the shadow compression ratio more
		// closely reflect the ultimate compression ratio.
		if err = t.shadowCompress.Flush(); err != nil {
			return 0, err
		}
		newBound = uint64(t.shadowBuf.Len()) + CloseOverheadZlib
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

	if t.compressAlgo == "brotli" {
		return t.brotliCompress.Write(p)
	} else if t.compressAlgo == "zstd" {
		return t.zstdCompress.Write(p)
	}
	return t.compress.Write(p)
}

func (t *ShadowCompressor) Close() error {
	if t.compressAlgo == "brotli" {
		return t.brotliCompress.Close()
	} else if t.compressAlgo == "zstd" {
		return t.zstdCompress.Close()
	}
	return t.compress.Close()
}

func (t *ShadowCompressor) Read(p []byte) (int, error) {
	if t.compressAlgo == "brotli" {
		return t.brotliBuf.Read(p)
	} else if t.compressAlgo == "zstd" {
		return t.zstdBuf.Read(p)
	}

	return t.zlibBuf.Read(p)
}

func (t *ShadowCompressor) Reset() {
	if t.compressAlgo == "brotli" {
		t.brotliBuf.Reset()
		t.brotliCompress.Reset(&t.brotliBuf)
	} else if t.compressAlgo == "zstd" {
		// no reset for zstd, so initialize new compressor instead
		t.zstdCompress = zstd.NewWriterLevel(&t.zstdBuf, 22)
	} else {
		t.compress.Reset(&t.zlibBuf)
	}
	t.shadowBuf.Reset()
	t.shadowCompress.Reset(&t.shadowBuf)
	t.fullErr = nil
	t.bound = safeCompressionOverhead
}

func (t *ShadowCompressor) Len() int {
	if t.compressAlgo == "brotli" {
		return t.brotliBuf.Len()
	} else if t.compressAlgo == "zstd" {
		return t.zstdBuf.Len()
	}
	return t.zlibBuf.Len()
}

func (t *ShadowCompressor) Flush() error {
	if t.compressAlgo == "brotli" {
		return t.brotliCompress.Flush()
	} else if t.compressAlgo == "zstd" {
		return t.zstdCompress.Flush()
	}
	return t.compress.Flush()
}

func (t *ShadowCompressor) FullErr() error {
	return t.fullErr
}

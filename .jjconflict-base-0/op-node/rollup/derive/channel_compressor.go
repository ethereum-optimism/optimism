package derive

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
)

const (
	ChannelVersionBrotli byte = 0x01
)

type ChannelCompressor interface {
	Write([]byte) (int, error)
	Flush() error
	Close() error
	Reset()
	Len() int
	// StaticBytesLen returns the number of header bytes written to the buffer at construction time
	StaticBytesLen() int
	Read([]byte) (int, error)
	GetCompressed() *bytes.Buffer
}

type CompressorWriter interface {
	Write([]byte) (int, error)
	Flush() error
	Close() error
	Reset(io.Writer)
}

type BaseChannelCompressor struct {
	compressed *bytes.Buffer
	CompressorWriter
}

func (bcc *BaseChannelCompressor) Len() int {
	return bcc.compressed.Len()
}

func (bcc *BaseChannelCompressor) Read(p []byte) (int, error) {
	return bcc.compressed.Read(p)
}

func (bcc *BaseChannelCompressor) GetCompressed() *bytes.Buffer {
	return bcc.compressed
}

type ZlibCompressor struct {
	BaseChannelCompressor
}

func (zc *ZlibCompressor) Reset() {
	zc.compressed.Reset()
	zc.CompressorWriter.Reset(zc.compressed)
}

func (bc *ZlibCompressor) StaticBytesLen() int {
	return 0
}

type BrotliCompressor struct {
	BaseChannelCompressor
}

func (bc *BrotliCompressor) Reset() {
	bc.compressed.Reset()
	bc.compressed.WriteByte(ChannelVersionBrotli)
	bc.CompressorWriter.Reset(bc.compressed)
}

func (bc *BrotliCompressor) StaticBytesLen() int {
	return 1
}

func NewChannelCompressor(algo CompressionAlgo) (ChannelCompressor, error) {
	compressed := &bytes.Buffer{}
	if algo == Zlib {
		writer, err := zlib.NewWriterLevel(compressed, zlib.BestCompression)
		if err != nil {
			return nil, err
		}
		return &ZlibCompressor{
			BaseChannelCompressor{
				CompressorWriter: writer,
				compressed:       compressed,
			},
		}, nil
	} else if algo.IsBrotli() {
		compressed.WriteByte(ChannelVersionBrotli)
		writer := brotli.NewWriterLevel(compressed, GetBrotliLevel(algo))
		return &BrotliCompressor{
			BaseChannelCompressor{
				CompressorWriter: writer,
				compressed:       compressed,
			},
		}, nil
	} else {
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algo)
	}
}

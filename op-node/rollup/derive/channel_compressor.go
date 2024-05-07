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
	writer     CompressorWriter
}

func (bcc *BaseChannelCompressor) Write(data []byte) (int, error) {
	return bcc.writer.Write(data)
}

func (bcc *BaseChannelCompressor) Flush() error {
	return bcc.writer.Flush()
}

func (bcc *BaseChannelCompressor) Close() error {
	return bcc.writer.Close()
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
	zc.writer.Reset(zc.compressed)
}

type BrotliCompressor struct {
	BaseChannelCompressor
}

func (bc *BrotliCompressor) Reset() {
	bc.compressed.Reset()
	bc.compressed.WriteByte(ChannelVersionBrotli)
	bc.writer.Reset(bc.compressed)
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
				writer:     writer,
				compressed: compressed,
			},
		}, nil
	} else if algo.IsBrotli() {
		compressed.WriteByte(ChannelVersionBrotli)
		writer := brotli.NewWriterLevel(compressed, GetBrotliLevel(algo))
		return &BrotliCompressor{
			BaseChannelCompressor{
				writer:     writer,
				compressed: compressed,
			},
		}, nil
	} else {
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algo)
	}
}

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

type CompressorWriter interface {
	Write([]byte) (int, error)
	Flush() error
	Close() error
	Reset(io.Writer)
}

type ChannelCompressor struct {
	writer          CompressorWriter
	compressionAlgo CompressionAlgo
	compressed      *bytes.Buffer
}

func NewChannelCompressor(algo CompressionAlgo) (*ChannelCompressor, error) {
	var writer CompressorWriter
	var err error
	compressed := &bytes.Buffer{}
	if algo == Zlib {
		writer, err = zlib.NewWriterLevel(compressed, zlib.BestCompression)
	} else if algo.IsBrotli() {
		compressed.WriteByte(ChannelVersionBrotli)
		writer = brotli.NewWriterLevel(compressed, GetBrotliLevel(algo))
	} else {
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algo)
	}

	if err != nil {
		return nil, err
	}

	return &ChannelCompressor{
		writer:          writer,
		compressionAlgo: algo,
		compressed:      compressed,
	}, nil

}

func (cc *ChannelCompressor) Write(data []byte) (int, error) {
	return cc.writer.Write(data)
}

func (cc *ChannelCompressor) Flush() error {
	return cc.writer.Flush()
}

func (cc *ChannelCompressor) Close() error {
	return cc.writer.Close()
}

func (cc *ChannelCompressor) Reset() {
	cc.compressed.Reset()
	if cc.compressionAlgo.IsBrotli() {
		// always add channal version for brotli
		cc.compressed.WriteByte(ChannelVersionBrotli)
	}
	cc.writer.Reset(cc.compressed)
}

func (cc *ChannelCompressor) Len() int {
	return cc.compressed.Len()
}

func (cc *ChannelCompressor) GetCompressed() *bytes.Buffer {
	return cc.compressed
}

func (cc *ChannelCompressor) Read(p []byte) (int, error) {
	return cc.compressed.Read(p)
}

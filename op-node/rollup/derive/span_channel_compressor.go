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

type SpanChannelCompressor struct {
	writer          CompressorWriter
	compressionAlgo CompressionAlgo
	compressed      *bytes.Buffer
}

func NewSpanChannelCompressor(algo CompressionAlgo) (*SpanChannelCompressor, error) {
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

	return &SpanChannelCompressor{
		writer:          writer,
		compressionAlgo: algo,
		compressed:      compressed,
	}, nil

}

func (scc *SpanChannelCompressor) Write(data []byte) (int, error) {
	return scc.writer.Write(data)
}

func (scc *SpanChannelCompressor) Flush() error {
	return scc.writer.Flush()
}

func (scc *SpanChannelCompressor) Close() error {
	return scc.writer.Close()
}

func (scc *SpanChannelCompressor) Reset() {
	scc.compressed.Reset()
	if scc.compressionAlgo.IsBrotli() {
		// always add channal version for brotli
		scc.compressed.WriteByte(ChannelVersionBrotli)
	}
	scc.writer.Reset(scc.compressed)
}

func (scc *SpanChannelCompressor) GetCompressedLen() int {
	return scc.compressed.Len()
}

func (scc *SpanChannelCompressor) GetCompressed() *bytes.Buffer {
	return scc.compressed
}

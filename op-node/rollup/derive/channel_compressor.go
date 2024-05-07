package derive

import (
	"bytes"
	"compress/zlib"
	"fmt"

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

type ZlibCompressor struct {
	writer          *zlib.Writer
	compressed      *bytes.Buffer
}

func (zc *ZlibCompressor) Write(data []byte) (int, error) {
	return zc.writer.Write(data)
}

func (zc *ZlibCompressor) Flush() error {
	return zc.writer.Flush()
}

func (zc *ZlibCompressor) Close() error {
	return zc.writer.Close()
}

func (zc *ZlibCompressor) Reset() {
	zc.compressed.Reset()
	zc.writer.Reset(zc.compressed)
}

func (zc *ZlibCompressor) Len() int {
	return zc.compressed.Len()
}

func (zc *ZlibCompressor) Read(p []byte) (int, error) {
	return zc.compressed.Read(p)
}

func (zc *ZlibCompressor) GetCompressed() *bytes.Buffer {
	return zc.compressed
}

type BrotliCompressor struct {
	writer          *brotli.Writer
	compressed      *bytes.Buffer
}

func (bc *BrotliCompressor) Write(data []byte) (int, error) {
	return bc.writer.Write(data)
}

func (bc *BrotliCompressor) Flush() error {
	return bc.writer.Flush()
}

func (bc *BrotliCompressor) Close() error {
	return bc.writer.Close()
}

func (bc *BrotliCompressor) Len() int {
	return bc.compressed.Len()
}

func (bc *BrotliCompressor) Read(p []byte) (int, error) {
	return bc.compressed.Read(p)
}

func (bc *BrotliCompressor) Reset() {
	bc.compressed.Reset()
	bc.compressed.WriteByte(ChannelVersionBrotli)
	bc.writer.Reset(bc.compressed)
}

func (bc *BrotliCompressor) GetCompressed() *bytes.Buffer {
	return bc.compressed
}

func NewChannelCompressor(algo CompressionAlgo) (ChannelCompressor, error) {
	compressed := &bytes.Buffer{}
	if algo == Zlib {
		writer, err := zlib.NewWriterLevel(compressed, zlib.BestCompression)
		if err != nil {
			return nil, err
		}
		return &ZlibCompressor{
			writer:          writer,
			compressed:      compressed,
		}, nil
	} else if algo.IsBrotli() {
		compressed.WriteByte(ChannelVersionBrotli)
		writer := brotli.NewWriterLevel(compressed, GetBrotliLevel(algo))
		return &BrotliCompressor{
			writer:          writer,
			compressed:      compressed,
		}, nil
	} else {
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algo)
	}
}

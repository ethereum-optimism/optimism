package ioutil

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// OpenDecompressed opens a reader for the specified file and automatically gzip decompresses the content
// if the filename ends with .gz
func OpenDecompressed(path string) (io.ReadCloser, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if IsGzip(path) {
		gr, err := gzip.NewReader(r)
		if err != nil {
			r.Close()
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return NewWrappedReadCloser(gr, r), nil
	}
	return r, nil
}

// OpenCompressed opens a file for writing and automatically compresses the content if the filename ends with .gz
func OpenCompressed(file string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	out, err := os.OpenFile(file, flag, perm)
	if err != nil {
		return nil, err
	}
	return CompressByFileType(file, out), nil
}

// WriteCompressedBytes writes a byte slice to the specified file.
// If the filename ends with .gz, a byte slice is compressed and written.
func WriteCompressedBytes(file string, data []byte, flag int, perm os.FileMode) error {
	out, err := OpenCompressed(file, flag, perm)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = out.Write(data)
	return err
}

// WriteCompressedJson writes the object to the specified file as a compressed json object
// if the filename ends with .gz.
func WriteCompressedJson(file string, obj any) error {
	if !IsGzip(file) {
		return fmt.Errorf("file %v does not have .gz extension", file)
	}
	out, err := OpenCompressed(file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	return json.NewEncoder(out).Encode(obj)
}

// IsGzip determines if a path points to a gzip compressed file.
// Returns true when the file has a .gz extension.
func IsGzip(path string) bool {
	return strings.HasSuffix(path, ".gz")
}

func CompressByFileType(file string, out io.WriteCloser) io.WriteCloser {
	if IsGzip(file) {
		return NewWrappedWriteCloser(gzip.NewWriter(out), out)
	}
	return out
}

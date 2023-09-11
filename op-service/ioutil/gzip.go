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
	var r io.ReadCloser
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if IsGzip(path) {
		r, err = gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
	}
	return r, nil
}

// OpenCompressed opens a file for writing and automatically compresses the content if the filename ends with .gz
func OpenCompressed(file string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	var out io.WriteCloser
	out, err := os.OpenFile(file, flag, perm)
	if err != nil {
		return nil, err
	}
	if IsGzip(file) {
		out = gzip.NewWriter(out)
	}
	return out, nil
}

// WriteCompressedJson writes the object to the specified file as a compressed json object
// if the filename ends with .gz.
// If no flags are provided, [os.O_CREATE|os.O_WRONLY] is used.
func WriteCompressedJson(file string, obj any, flags ...int) error {
	if !IsGzip(file) {
		return fmt.Errorf("file %v does not have .gz extension", file)
	}
	if len(flags) == 0 {
		flags = []int{os.O_CREATE | os.O_WRONLY}
	}
	var flag int
	for _, arg := range flags {
		flag |= arg
	}
	out, err := OpenCompressed(file, flag, 0644)
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

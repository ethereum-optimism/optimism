// Probes whether andybalholm/brotli reports clean EOF on truncated streams,
// which would be a spec deviation per RFC 7932.
package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
)

func main() {
	// Compress a known plaintext.
	plaintext := []byte("the quick brown fox jumps over the lazy dog twice for redundancy and length")
	var buf bytes.Buffer
	w := brotli.NewWriter(&buf)
	_, _ = w.Write(plaintext)
	_ = w.Close()
	compressed := buf.Bytes()
	fmt.Printf("plaintext: %d bytes\n", len(plaintext))
	fmt.Printf("compressed: %d bytes (%s)\n", len(compressed), hex.EncodeToString(compressed))

	fmt.Printf("\n%-12s %-15s %-9s %-9s %s\n", "trunc_at", "result", "out_bytes", "input_left", "err")
	fmt.Println("------------------------------------------------------------")
	for i := len(compressed) - 1; i >= 1; i-- {
		truncated := compressed[:i]
		src := bytes.NewReader(truncated)
		br := brotli.NewReader(src)

		output, err := io.ReadAll(br)
		errStr := "<nil>"
		if err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				errStr = "ErrUnexpectedEOF"
			} else if errors.Is(err, io.EOF) {
				errStr = "EOF"
			} else {
				errStr = err.Error()
			}
		}
		flag := ""
		if err == nil {
			flag = "  ⚠ accepted truncated stream"
		}
		fmt.Printf("%-12d %-15s %-9d %-9d %s%s\n",
			i, errStr, len(output), src.Len(), errStr, flag)
		_ = output
	}
}

// Probes whether google/brotli (cgo to reference C) reports clean EOF on
// truncated streams. If argv[1] is provided, it's a hex-encoded compressed
// brotli stream; otherwise the program compresses a default plaintext.
package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/brotli/go/cbrotli"
)

func main() {
	var compressed []byte
	if len(os.Args) > 1 {
		var err error
		compressed, err = hex.DecodeString(os.Args[1])
		if err != nil {
			panic(err)
		}
	} else {
		plaintext := []byte("the quick brown fox jumps over the lazy dog twice for redundancy and length")
		var buf bytes.Buffer
		w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 5})
		_, _ = w.Write(plaintext)
		_ = w.Close()
		compressed = buf.Bytes()
		fmt.Printf("plaintext: %d bytes\n", len(plaintext))
	}
	fmt.Printf("compressed: %d bytes (%s)\n", len(compressed), hex.EncodeToString(compressed))

	fmt.Printf("\n%-10s %-25s %-10s %-10s\n", "trunc_at", "result", "out_bytes", "input_left")
	for i := len(compressed) - 1; i >= 1; i-- {
		truncated := compressed[:i]
		src := bytes.NewReader(truncated)
		br := cbrotli.NewReader(src)

		output, err := io.ReadAll(br)
		_ = br.Close()
		errStr := "Ok"
		if err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				errStr = "ErrUnexpectedEOF"
			} else if errors.Is(err, io.EOF) {
				errStr = "EOF"
			} else {
				errStr = err.Error()
			}
		}
		fmt.Printf("%-10d %-25s %-10d %-10d\n", i, errStr, len(output), src.Len())
	}
}

// Reads hex-encoded plaintext from argv[1], compresses with cbrotli, and
// prints the compressed bytes as hex on stdout.
package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/google/brotli/go/cbrotli"
)

func main() {
	in, err := hex.DecodeString(os.Args[1])
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 5})
	if _, err := w.Write(in); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	fmt.Println(hex.EncodeToString(buf.Bytes()))
}

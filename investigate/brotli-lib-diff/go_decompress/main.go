// Reads hex-encoded brotli-compressed bytes from stdin, decompresses with the
// requested implementation, and prints:
//
//	STATUS=ok|err
//	ERR=<error message if err>
//	OUTPUT=<hex>
//
// Selection: --impl=andybalholm (default) or --impl=cbrotli.
package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/google/brotli/go/cbrotli"
)

func main() {
	implFlag := flag.String("impl", "andybalholm", "brotli implementation: andybalholm or cbrotli")
	flag.Parse()

	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	compressed, err := hex.DecodeString(strings.TrimSpace(string(in)))
	if err != nil {
		panic(err)
	}

	var out []byte
	var derr error
	switch *implFlag {
	case "andybalholm":
		r := brotli.NewReader(bytes.NewReader(compressed))
		out, derr = io.ReadAll(r)
	case "cbrotli":
		r := cbrotli.NewReader(bytes.NewReader(compressed))
		out, derr = io.ReadAll(r)
		_ = r.Close()
	default:
		fmt.Fprintf(os.Stderr, "unknown impl %q\n", *implFlag)
		os.Exit(2)
	}

	if derr == nil {
		fmt.Println("STATUS=ok")
	} else {
		fmt.Println("STATUS=err")
		fmt.Printf("ERR=%s\n", derr.Error())
	}
	fmt.Printf("OUTPUT=%s\n", hex.EncodeToString(out))
}

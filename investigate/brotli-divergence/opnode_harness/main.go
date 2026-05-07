// Reads hex-encoded channel bytes from stdin and emits op-node's behavior:
//
//	DECOMPRESS_RESULT=ok|err
//	DECOMPRESS_BYTES=<hex>           (bytes that brotli.NewReader produced before EOF/err)
//	BATCH_COUNT=<n>                  (batches accepted by full BatchReader pipeline)
//
// The input must start with the channel version byte (0x01 for brotli);
// the rest is the brotli-compressed payload.
package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

const maxRLPBytesPerChannelFjord = 100_000_000

func main() {
	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	channel, err := hex.DecodeString(strings.TrimSpace(string(in)))
	if err != nil {
		panic(err)
	}
	if len(channel) == 0 {
		panic("empty channel")
	}
	if channel[0] != 0x01 {
		panic("channel version must be 0x01 (brotli)")
	}

	// 1) Raw brotli decompression: read everything brotli is willing to give us.
	//    Brotli's Reader.Read returns (n, err) — when the stream is truncated, the
	//    final Read returns the leftover bytes followed by io.ErrUnexpectedEOF (or
	//    similar). io.ReadAll concatenates all successful reads and surfaces the
	//    final error, so it gives us "bytes produced before failure".
	br := brotli.NewReader(bytes.NewReader(channel[1:]))
	rawBytes, rawErr := io.ReadAll(br)
	if rawErr == nil {
		fmt.Println("DECOMPRESS_RESULT=ok")
	} else {
		fmt.Println("DECOMPRESS_RESULT=err")
	}
	fmt.Printf("DECOMPRESS_BYTES=%s\n", hex.EncodeToString(rawBytes))

	// 1b) Strict variant (Seb's proposed rule):
	//   - reach clean io.EOF, AND no leftover input bytes,
	//   - OR produce exactly maxRLPBytesPerChannelFjord bytes (zip-bomb cap),
	//   - else reject.
	strictResult, strictBytes := strictDecompress(channel[1:], maxRLPBytesPerChannelFjord)
	if strictResult {
		fmt.Println("STRICT_RESULT=ok")
		fmt.Printf("STRICT_BYTES=%s\n", hex.EncodeToString(strictBytes))
	} else {
		fmt.Println("STRICT_RESULT=err")
		fmt.Println("STRICT_BYTES=")
	}

	// 2) Full BatchReader pipeline: how many batches op-node accepts.
	//    isFjord=true so the brotli gate is open.
	reader, err := derive.BatchReader(bytes.NewReader(channel), maxRLPBytesPerChannelFjord, true)
	if err != nil {
		fmt.Println("BATCH_COUNT=0")
		return
	}
	count := 0
	for {
		_, err := reader()
		if err != nil {
			break
		}
		count++
	}
	fmt.Printf("BATCH_COUNT=%d\n", count)
}

// strictDecompress returns (true, bytes) if either:
//   - the brotli stream ended cleanly (io.EOF) AND all input was consumed, OR
//   - exactly `max` output bytes were produced (zip-bomb cap).
//
// Otherwise returns (false, nil).
func strictDecompress(input []byte, max int) (bool, []byte) {
	src := bytes.NewReader(input)
	br := brotli.NewReader(src)

	// Read up to max+1 bytes to detect the cap case unambiguously.
	limited := io.LimitReader(br, int64(max)+1)
	output, err := io.ReadAll(limited)
	if err != nil {
		// Brotli error mid-stream → reject.
		return false, nil
	}

	if len(output) > max {
		// Future-fork rule: cap exceeded means the channel is invalid — reject entirely.
		return false, nil
	}

	// Read returned <= max bytes with no error. Verify clean EOF: one more read
	// must yield (0, io.EOF). Anything else (e.g., bytes available, or a non-EOF
	// error) means the stream did not end cleanly.
	tail := make([]byte, 1)
	n, err := br.Read(tail)
	if n != 0 || err != io.EOF {
		return false, nil
	}

	// And no leftover input bytes (post-stream garbage).
	if src.Len() != 0 {
		return false, nil
	}

	return true, output
}

// Generates a brotli-compressed channel containing N test SingularBatches.
// Prints the channel bytes as hex on stdout.
package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

func main() {
	n := flag.Int("n", 3, "number of batches")
	flag.Parse()

	var rlpBuf bytes.Buffer
	for i := 0; i < *n; i++ {
		sb := &derive.SingularBatch{
			ParentHash: common.HexToHash(fmt.Sprintf("0x%064x", i+1)),
			EpochNum:   rollup.Epoch(100 + i),
			EpochHash:  common.HexToHash(fmt.Sprintf("0x%064x", 0xAA00+i)),
			Timestamp:  uint64(1_700_000_000 + i*2),
			Transactions: []hexutil.Bytes{
				[]byte(fmt.Sprintf("tx-%d-A", i)),
				[]byte(fmt.Sprintf("tx-%d-B", i)),
			},
		}
		bd := derive.NewBatchData(sb)
		if err := rlp.Encode(&rlpBuf, bd); err != nil {
			panic(err)
		}
	}

	var compressed bytes.Buffer
	w := brotli.NewWriter(&compressed)
	if _, err := w.Write(rlpBuf.Bytes()); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}

	out := []byte{0x01}
	out = append(out, compressed.Bytes()...)
	fmt.Println(hex.EncodeToString(out))
}

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func PreCanyonEncode(receipts types.Receipts) [][]byte {
	for _, receipt := range receipts {
		if receipt.Type == types.DepositTxType {
			receipt.DepositReceiptVersion = nil
		}
	}
	var out [][]byte
	for i := range receipts {
		var buf bytes.Buffer
		receipts.EncodeIndex(i, &buf)
		out = append(out, buf.Bytes())
	}
	return out
}

func PostCanyonEncode(receipts types.Receipts) [][]byte {
	v := uint64(1)
	for _, receipt := range receipts {
		if receipt.Type == types.DepositTxType {
			receipt.DepositReceiptVersion = &v
		}
	}
	var out [][]byte
	for i := range receipts {
		var buf bytes.Buffer
		receipts.EncodeIndex(i, &buf)
		out = append(out, buf.Bytes())
	}
	return out
}

func HashList(list [][]byte) common.Hash {
	t := trie.NewEmpty(trie.NewDatabase(rawdb.NewDatabase(memorydb.New()), nil))
	for i, value := range list {
		var index []byte
		val := make([]byte, len(value))
		copy(val, value)
		index = rlp.AppendUint64(index, uint64(i))
		if err := t.Update(index, val); err != nil {
			panic(err)
		}
	}
	return t.Hash()
}

type ReceiptFetcher interface {
	InfoByNumber(context.Context, uint64) (eth.BlockInfo, error)
	FetchReceipts(context.Context, common.Hash) (eth.BlockInfo, types.Receipts, error)
}

func ValidatePreCanyon(number uint64, client ReceiptFetcher) error {
	block, err := client.InfoByNumber(context.Background(), number)
	if err != nil {
		return err
	}
	_, receipts, err := client.FetchReceipts(context.Background(), block.Hash())
	if err != nil {
		return err
	}

	have := block.ReceiptHash()
	want := HashList(PreCanyonEncode(receipts))
	if have != want {
		return fmt.Errorf("Receipts do not look correct as pre-canyon. have: %v, want: %v", have, want)
	}
	return nil
}

func ValidatePostCanyon(number uint64, client ReceiptFetcher) error {
	block, err := client.InfoByNumber(context.Background(), number)
	if err != nil {
		return err
	}
	_, receipts, err := client.FetchReceipts(context.Background(), block.Hash())
	if err != nil {
		return err
	}

	have := block.ReceiptHash()
	want := HashList(PostCanyonEncode(receipts))
	if have != want {
		return fmt.Errorf("Receipts do not look correct as post-canyon. have: %v, want: %v", have, want)
	}
	return nil
}

func main() {
	logger := log.New()

	// Define the flag variables
	var (
		preCanyon bool
		number    uint64
		rpcURL    string
	)

	// Define and parse the command-line flags
	flag.BoolVar(&preCanyon, "pre-canyon", false, "Set this flag to assert pre-canyon receipt hash behavior")
	flag.Uint64Var(&number, "number", 0, "Specify a uint64 value for number")
	flag.StringVar(&rpcURL, "rpc-url", "", "Specify the RPC URL as a string")

	// Parse the command-line arguments
	flag.Parse()

	l1RPC, err := client.NewRPC(context.Background(), logger, rpcURL, client.WithDialBackoff(10))
	if err != nil {
		log.Crit("Error creating RPC", "err", err)
	}
	c := &rollup.Config{SeqWindowSize: 10}
	l1ClCfg := sources.L1ClientDefaultConfig(c, true, sources.RPCKindBasic)
	client, err := sources.NewL1Client(l1RPC, logger, nil, l1ClCfg)
	if err != nil {
		log.Crit("Error creating RPC", "err", err)
	}

	if preCanyon {
		if err := ValidatePreCanyon(number, client); err != nil {
			log.Crit("Pre Canyon should succeed when expecting pre-canyon", "err", err)
		}
		if err := ValidatePostCanyon(number, client); err == nil {
			log.Crit("Post Canyon should fail when expecting pre-canyon")
		}
	} else {
		if err := ValidatePostCanyon(number, client); err != nil {
			log.Crit("Post Canyon should succeed when expecting post-canyon", "err", err)
		}
		if err := ValidatePreCanyon(number, client); err == nil {
			log.Crit("Pre Canyon should fail when expecting post-canyon")
		}
	}

}

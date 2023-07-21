package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"

	openrpc "github.com/rollkit/celestia-openrpc"
	"github.com/rollkit/celestia-openrpc/types/share"
)

func main() {
	if len(os.Args) < 4 {
		panic("usage: op-celestia <namespace> <eth calldata> <auth token>")
	}
	data, _ := hex.DecodeString(os.Args[2])
	buf := bytes.NewBuffer(data)
	var height int64
	err := binary.Read(buf, binary.BigEndian, &height)
	if err != nil {
		panic(err)
	}
	var index uint32
	err = binary.Read(buf, binary.BigEndian, &index)
	if err != nil {
		panic(err)
	}
	fmt.Printf("celestia block height: %v; tx index: %v\n", height, index)
	fmt.Println("-----------------------------------------")
	client, err := openrpc.NewClient(context.Background(), "http://localhost:26658", os.Args[3])
	if err != nil {
		panic(err)
	}
	nsBytes, err := hex.DecodeString(os.Args[1])
	if err != nil {
		panic(err)
	}
	namespace, err := share.NewBlobNamespaceV0(nsBytes)
	if err != nil {
		panic(err)
	}

	namespacedData, err := client.Blob.GetAll(context.Background(), uint64(height), []share.Namespace{namespace})
	if err != nil {
		panic(err)
	}
	fmt.Printf("optimism block data on celestia: %x\n", namespacedData[0].Data)
}

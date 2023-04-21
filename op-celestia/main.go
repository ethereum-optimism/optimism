package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/celestiaorg/go-cnc"
)

func main() {
	data, _ := hex.DecodeString(os.Args[1])
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
	client, err := cnc.NewClient("http://localhost:26659", cnc.WithTimeout(30*time.Second))
	if err != nil {
		panic(err)
	}
	namespaceId, _ := hex.DecodeString("e8e5f679bf7116cbe5f679ef")
	var nid [8]byte
	copy(nid[:], namespaceId)
	namespacedData, err := client.NamespacedData(context.Background(), nid, uint64(height))
	if err != nil {
		panic(err)
	}
	fmt.Printf("optimism block data on celestia: %x\n", namespacedData)
}

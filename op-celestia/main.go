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
	client, err := cnc.NewClient("http://localhost:26659", cnc.WithTimeout(30*time.Second))
	if err != nil {
		panic(err)
	}
	nsBytes, err := hex.DecodeString(os.Args[1])
	if err != nil {
		panic(err)
	}
	namespace := cnc.MustNewV0(nsBytes)
	namespacedData, err := client.NamespacedData(context.Background(), namespace, uint64(height))
	if err != nil {
		panic(err)
	}
	fmt.Printf("optimism block data on celestia: %x\n", namespacedData)
}

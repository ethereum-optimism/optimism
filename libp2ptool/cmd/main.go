package main

import (
	"flag"
	"fmt"
	"libp2ptool"
	"os"
)

var isPrivateKey bool

func main() {
	flag.Parse()
	key, err := libp2ptool.ReadPeerID(isPrivateKey, os.Stdin)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println(key)
}

func init() {
	flag.BoolVar(&isPrivateKey, "private-key", false, "whether the input is a private key")
}

package main

import (
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/ethereum-optimism/optimism/proxyd/tools/mockserver/handler"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("simply mock a response based on an external text MockedResponsesFile\n")
		fmt.Printf("usage: mockserver <port> <MockedResponsesFile.yml>\n")
		os.Exit(1)
	}
	port, _ := strconv.ParseInt(os.Args[1], 10, 32)
	dir, _ := os.Getwd()

	h := handler.MockedHandler{
		Autoload:     true,
		AutoloadFile: path.Join(dir, os.Args[2]),
	}

	err := h.Serve(int(port))
	if err != nil {
		fmt.Printf("error starting mockserver: %v\n", err)
	}
}

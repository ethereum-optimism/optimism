//go:debug decoratemappings=0
package main

import (
	"github.com/ethereum-optimism/optimism/op-program/client"
)

func main() {
	client.Main(true)
}

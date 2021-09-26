package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func main() {
	fmt.Println("hello")

	var parent types.Header
	database := state.NewDatabase(parent)
	statedb, _ := state.New(parent.Root, database, nil)

	config := vm.Config{}
	vm := vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, params.MainnetChainConfig, config)
	fmt.Println(vm)
}

package main

import (
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func main() {
	var scalar, blobScalar uint
	flag.UintVar(&scalar, "scalar", 0, "base fee scalar value for the gas config (uint32)")
	flag.UintVar(&blobScalar, "blob-scalar", 0, "blob base fee scalar value for the gas config (uint32)")
	flag.Parse()

	if scalar > math.MaxUint32 {
		fmt.Fprintln(flag.CommandLine.Output(), "scalar out of uint32 range")
		flag.Usage()
		os.Exit(2)
	}
	if blobScalar > math.MaxUint32 {
		fmt.Fprintln(flag.CommandLine.Output(), "blob-scalar out of uint32 range")
		flag.Usage()
		os.Exit(2)
	}

	encoded := eth.EncodeScalar(eth.EcostoneScalars{
		BlobBaseFeeScalar: uint32(blobScalar),
		BaseFeeScalar:     uint32(scalar),
	})
	i := new(big.Int).SetBytes(encoded[:])

	fmt.Println("# base fee scalar     :", scalar)
	fmt.Println("# blob base fee scalar:", blobScalar)
	fmt.Printf("# v1 hex encoding  : 0x%x\n", encoded[:])
	fmt.Println("# uint value for the 'scalar' parameter in SystemConfigProxy.setGasConfig():")
	fmt.Println(i)
}

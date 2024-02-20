package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
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

	var n [32]byte
	n[0] = 1 // version
	binary.BigEndian.PutUint32(n[32-4:], uint32(scalar))
	binary.BigEndian.PutUint32(n[32-8:], uint32(blobScalar))
	i := new(big.Int).SetBytes(n[:])

	fmt.Println("# base fee scalar     :", scalar)
	fmt.Println("# blob base fee scalar:", blobScalar)
	fmt.Printf("# v1 hex encoding  : 0x%x\n", n[:])
	fmt.Println("# uint value for the 'scalar' parameter in SystemConfigProxy.setGasConfig():")
	fmt.Println(i)
}

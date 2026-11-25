// Command parityreport prints the derivation parity report for a directory of parity records.
//
//	go run ./op-service/testutils/parityreport/cmd <dir>
package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/testutils/parityreport"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: parityreport <records dir>")
		os.Exit(2)
	}
	records, err := parityreport.Load(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(records) == 0 {
		fmt.Fprintf(os.Stderr, "no parity records in %s\n", os.Args[1])
		os.Exit(1)
	}
	fmt.Print(parityreport.Markdown(records))
}

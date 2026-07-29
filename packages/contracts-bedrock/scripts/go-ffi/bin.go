package main

import (
	"log"
	"os"
)

func main() {
	switch os.Args[1] {
	case "diff":
		DiffTestUtils()
	case "trie":
		FuzzTrie()
	case "merkle":
		DiffMerkle()
	case "opcm":
		GetOPCMs()
	case "nut-bundles":
		GetNUTBundles()
	default:
		log.Fatal("Must pass a subcommand")
	}
}

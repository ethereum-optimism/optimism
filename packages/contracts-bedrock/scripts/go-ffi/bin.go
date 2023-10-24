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
	default:
		log.Fatal("Must pass a subcommand")
	}
}

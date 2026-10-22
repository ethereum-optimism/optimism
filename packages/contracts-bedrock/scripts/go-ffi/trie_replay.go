package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	randv2 "math/rand/v2"
	"strings"
)

// Coverage replay controls only this test helper's entropy. Ordinary calls keep
// crypto/rand; a changed benchmark seed or argument list selects a new corpus.
var trieReplayEntropy io.Reader

func trieReplayReader(seed string, args []string, ci, profile string) (io.Reader, error) {
	if seed == "" {
		return nil, nil
	}
	if ci != "true" || profile != "cicoverage" {
		return nil, fmt.Errorf("trie replay requires CI coverage")
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(seed, "0x"))
	if err != nil || len(decoded) != sha256.Size {
		return nil, fmt.Errorf("trie replay requires a 256-bit benchmark seed")
	}
	encoded, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("encode trie replay arguments: %w", err)
	}
	material := append([]byte("optimism-contract-coverage-ffi-v1\x00"), decoded...)
	material = append(material, encoded...)
	return randv2.NewChaCha8(sha256.Sum256(material)), nil
}

func trieRead(p []byte) (int, error) {
	if trieReplayEntropy == nil {
		return rand.Read(p)
	}
	return io.ReadFull(trieReplayEntropy, p)
}

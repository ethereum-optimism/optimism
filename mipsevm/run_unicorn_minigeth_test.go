package main

import (
	"testing"
)

func TestMinigethUnicorn(t *testing.T) {
	uniram := make(map[uint32](uint32))
	RunUnicorn("../mipigo/minigeth.bin", uniram, -1, false, nil)
}

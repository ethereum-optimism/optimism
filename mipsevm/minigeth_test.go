package main

import (
	"testing"
)

func TestProfileMinigeth(t *testing.T) {
	interpreter, bytecode := GetInterpreterAndBytecode()
	RunMinigeth("../mipigeth/minigeth.bin", interpreter, bytecode, 400000)
}

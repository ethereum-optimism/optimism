package ssz

import "hash"

type HashFn func(dst []byte, input []byte) error

func NativeHashWrapper(hashFn hash.Hash) HashFn {
	return func(dst []byte, input []byte) error {
		hash := func(dst []byte, src []byte) {
			hashFn.Write(src[:32])
			hashFn.Write(src[32:64])
			hashFn.Sum(dst)
			hashFn.Reset()
		}

		layerLen := len(input) / 32
		if layerLen%2 == 1 {
			layerLen++
		}
		for i := 0; i < layerLen; i += 2 {
			hash(input[(i/2)*32:][:0], input[i*32:])
		}
		return nil
	}
}

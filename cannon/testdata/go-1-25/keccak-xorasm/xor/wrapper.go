package xor

func XORBytes(dst, a, b []byte) {
	n := min(len(a), len(b))
	xorBytes(&dst[0], &a[0], &b[0], n)
}

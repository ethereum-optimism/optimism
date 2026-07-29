package fees

// FlzCompressLen returns the length of the data after compression through FastLZ, based on
// https://github.com/Vectorized/solady/blob/5315d937d79b335c668896d7533ac603adac5315/js/solady.js
func FlzCompressLen(ib []byte) uint32 {
	n := uint32(0)
	ht := make([]uint32, 8192)
	u24 := func(i uint32) uint32 {
		return uint32(ib[i]) | (uint32(ib[i+1]) << 8) | (uint32(ib[i+2]) << 16)
	}
	cmp := func(p uint32, q uint32, e uint32) uint32 {
		l := uint32(0)
		for e -= q; l < e; l++ {
			if ib[p+l] != ib[q+l] {
				e = 0
			}
		}
		return l
	}
	literals := func(r uint32) {
		n += 0x21 * (r / 0x20)
		r %= 0x20
		if r != 0 {
			n += r + 1
		}
	}
	match := func(l uint32) {
		l--
		n += 3 * (l / 262)
		if l%262 >= 6 {
			n += 3
		} else {
			n += 2
		}
	}
	hash := func(v uint32) uint32 {
		return ((2654435769 * v) >> 19) & 0x1fff
	}
	setNextHash := func(ip uint32) uint32 {
		ht[hash(u24(ip))] = ip
		return ip + 1
	}
	a := uint32(0)
	ipLimit := uint32(len(ib)) - 13
	if len(ib) < 13 {
		ipLimit = 0
	}
	for ip := a + 2; ip < ipLimit; {
		r := uint32(0)
		d := uint32(0)
		for {
			s := u24(ip)
			h := hash(s)
			r = ht[h]
			ht[h] = ip
			d = ip - r
			if ip >= ipLimit {
				break
			}
			ip++
			if d <= 0x1fff && s == u24(r) {
				break
			}
		}
		if ip >= ipLimit {
			break
		}
		ip--
		if ip > a {
			literals(ip - a)
		}
		l := cmp(r+3, ip+3, ipLimit+9)
		match(l)
		ip = setNextHash(setNextHash(ip + l))
		a = ip
	}
	literals(uint32(len(ib)) - a)
	return n
}

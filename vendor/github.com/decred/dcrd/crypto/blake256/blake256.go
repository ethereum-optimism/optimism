// Copyright (c) 2019 The Decred developers
// Originally written in 2011-2012 by Dmitry Chestnykh.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package blake256 implements BLAKE-256 and BLAKE-224 hash functions (SHA-3
// candidate).
package blake256

import "hash"

// BlockSize is the block size of the hash algorithm in bytes.
const BlockSize = 64

// Size is the size of BLAKE-256 hash in bytes.
const Size = 32

// Size224 is the size of BLAKE-224 hash in bytes.
const Size224 = 28

type digest struct {
	hashSize int             // hash output size in bits (224 or 256)
	h        [8]uint32       // current chain value
	s        [4]uint32       // salt (zero by default)
	t        uint64          // message bits counter
	nullt    bool            // special case for finalization: skip counter
	x        [BlockSize]byte // buffer for data not yet compressed
	nx       int             // number of bytes in buffer
}

var (
	// Initialization values.
	iv256 = [8]uint32{
		0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A,
		0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19}

	iv224 = [8]uint32{
		0xC1059ED8, 0x367CD507, 0x3070DD17, 0xF70E5939,
		0xFFC00B31, 0x68581511, 0x64F98FA7, 0xBEFA4FA4}

	pad = [64]byte{0x80}
)

// Reset resets the state of digest. It leaves salt intact.
func (d *digest) Reset() {
	if d.hashSize == 224 {
		d.h = iv224
	} else {
		d.h = iv256
	}
	d.t = 0
	d.nx = 0
	d.nullt = false
}

func (d *digest) Size() int { return d.hashSize >> 3 }

func (d *digest) BlockSize() int { return BlockSize }

func (d *digest) Write(p []byte) (nn int, err error) {
	nn = len(p)
	if d.nx > 0 {
		n := len(p)
		if n > BlockSize-d.nx {
			n = BlockSize - d.nx
		}
		d.nx += copy(d.x[d.nx:], p)
		if d.nx == BlockSize {
			block(d, d.x[:])
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= BlockSize {
		n := len(p) &^ (BlockSize - 1)
		block(d, p[:n])
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

// Sum returns the calculated checksum.
func (d digest) Sum(in []byte) []byte {
	// Note d is a copy so that the caller can keep writing and summing.

	sum := d.checkSum()
	if d.Size() == Size224 {
		return append(in, sum[:Size224]...)
	}
	return append(in, sum[:]...)
}

func (d *digest) checkSum() [Size]byte {
	nx := uint64(d.nx)
	l := d.t + nx<<3
	var len [8]byte
	len[0] = byte(l >> 56)
	len[1] = byte(l >> 48)
	len[2] = byte(l >> 40)
	len[3] = byte(l >> 32)
	len[4] = byte(l >> 24)
	len[5] = byte(l >> 16)
	len[6] = byte(l >> 8)
	len[7] = byte(l)

	if nx == 55 {
		// One padding byte.
		d.t -= 8
		if d.hashSize == 224 {
			d.Write([]byte{0x80})
		} else {
			d.Write([]byte{0x81})
		}
	} else {
		if nx < 55 {
			// Enough space to fill the block.
			if nx == 0 {
				d.nullt = true
			}
			d.t -= 440 - nx<<3
			d.Write(pad[0 : 55-nx])
		} else {
			// Need 2 compressions.
			d.t -= 512 - nx<<3
			d.Write(pad[0 : 64-nx])
			d.t -= 440
			d.Write(pad[1:56])
			d.nullt = true
		}
		if d.hashSize == 224 {
			d.Write([]byte{0x00})
		} else {
			d.Write([]byte{0x01})
		}
		d.t -= 8
	}
	d.t -= 64
	d.Write(len[:])

	var out [Size]byte
	j := 0
	for _, s := range d.h[:d.hashSize>>5] {
		out[j+0] = byte(s >> 24)
		out[j+1] = byte(s >> 16)
		out[j+2] = byte(s >> 8)
		out[j+3] = byte(s >> 0)
		j += 4
	}
	return out
}

func (d *digest) setSalt(s []byte) {
	if len(s) != 16 {
		panic("salt length must be 16 bytes")
	}
	d.s[0] = uint32(s[0])<<24 | uint32(s[1])<<16 | uint32(s[2])<<8 | uint32(s[3])
	d.s[1] = uint32(s[4])<<24 | uint32(s[5])<<16 | uint32(s[6])<<8 | uint32(s[7])
	d.s[2] = uint32(s[8])<<24 | uint32(s[9])<<16 | uint32(s[10])<<8 | uint32(s[11])
	d.s[3] = uint32(s[12])<<24 | uint32(s[13])<<16 | uint32(s[14])<<8 | uint32(s[15])
}

// New returns a new hash.Hash computing the BLAKE-256 checksum.
func New() hash.Hash {
	return &digest{
		hashSize: 256,
		h:        iv256,
	}
}

// NewSalt is like New but initializes salt with the given 16-byte slice.
func NewSalt(salt []byte) hash.Hash {
	d := &digest{
		hashSize: 256,
		h:        iv256,
	}
	d.setSalt(salt)
	return d
}

// New224 returns a new hash.Hash computing the BLAKE-224 checksum.
func New224() hash.Hash {
	return &digest{
		hashSize: 224,
		h:        iv224,
	}
}

// New224Salt is like New224 but initializes salt with the given 16-byte slice.
func New224Salt(salt []byte) hash.Hash {
	d := &digest{
		hashSize: 224,
		h:        iv224,
	}
	d.setSalt(salt)
	return d
}

// Sum256 returns the BLAKE-256 checksum of the data.
func Sum256(data []byte) [Size]byte {
	var d digest
	d.hashSize = 256
	d.Reset()
	d.Write(data)
	return d.checkSum()
}

// Sum224 returns the BLAKE-224 checksum of the data.
func Sum224(data []byte) (sum224 [Size224]byte) {
	var d digest
	d.hashSize = 224
	d.Reset()
	d.Write(data)
	sum := d.checkSum()
	copy(sum224[:], sum[:Size224])
	return
}

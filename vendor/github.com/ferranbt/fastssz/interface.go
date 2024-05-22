package ssz

// Marshaler is the interface implemented by types that can marshal themselves into valid SZZ.
type Marshaler interface {
	MarshalSSZTo(dst []byte) ([]byte, error)
	MarshalSSZ() ([]byte, error)
	SizeSSZ() int
}

// Unmarshaler is the interface implemented by types that can unmarshal a SSZ description of themselves
type Unmarshaler interface {
	UnmarshalSSZ(buf []byte) error
}

type HashRoot interface {
	GetTree() (*Node, error)
	HashTreeRoot() ([32]byte, error)
	HashTreeRootWith(hh HashWalker) error
}

type HashWalker interface {
	// Intended for testing purposes to know the latest hash generated during merkleize
	Hash() []byte
	AppendUint8(i uint8)
	AppendUint64(i uint64)
	AppendBytes32(b []byte)
	PutUint64(i uint64)
	PutUint32(i uint32)
	PutUint16(i uint16)
	PutUint8(i uint8)
	FillUpTo32()
	Append(i []byte)
	PutBitlist(bb []byte, maxSize uint64)
	PutBool(b bool)
	PutBytes(b []byte)
	Index() int
	Merkleize(indx int)
	MerkleizeWithMixin(indx int, num, limit uint64)
}

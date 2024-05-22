package ssz

var _ HashWalker = (*Wrapper)(nil)

// ProofTree hashes a HashRoot object with a Hasher from
// the default HasherPool
func ProofTree(v HashRoot) (*Node, error) {
	w := &Wrapper{}
	if err := v.HashTreeRootWith(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

type Wrapper struct {
	nodes []*Node
	buf   []byte
}

/// --- wrapper implements the HashWalker interface ---

func (w *Wrapper) Index() int {
	return len(w.nodes)
}

func (w *Wrapper) Append(i []byte) {
	w.buf = append(w.buf, i...)
}

func (w *Wrapper) AppendUint64(i uint64) {
	w.buf = MarshalUint64(w.buf, i)
}

func (w *Wrapper) AppendUint8(i uint8) {
	w.buf = MarshalUint8(w.buf, i)
}

func (w *Wrapper) AppendBytes32(b []byte) {
	w.buf = append(w.buf, b...)
	w.FillUpTo32()
}

func (w *Wrapper) FillUpTo32() {
	// pad zero bytes to the left
	if rest := len(w.buf) % 32; rest != 0 {
		w.buf = append(w.buf, zeroBytes[:32-rest]...)
	}
}

func (w *Wrapper) Merkleize(indx int) {
	if len(w.buf) != 0 {
		w.appendBytesAsNodes(w.buf)
		w.buf = w.buf[:0]
	}
	w.Commit(indx)
}

func (w *Wrapper) MerkleizeWithMixin(indx int, num, limit uint64) {
	if len(w.buf) != 0 {
		w.appendBytesAsNodes(w.buf)
		w.buf = w.buf[:0]
	}
	w.CommitWithMixin(indx, int(num), int(limit))
}

func (w *Wrapper) PutBitlist(bb []byte, maxSize uint64) {
	b, size := parseBitlist(nil, bb)

	indx := w.Index()
	w.appendBytesAsNodes(b)
	w.CommitWithMixin(indx, int(size), int((maxSize+255)/256))
}

func (w *Wrapper) appendBytesAsNodes(b []byte) {
	for i := 0; i < len(b); i += 32 {
		val := append([]byte{}, b[i:min(len(b), i+32)]...)
		w.nodes = append(w.nodes, LeafFromBytes(val))
	}
}

func (w *Wrapper) PutBool(b bool) {
	w.AddNode(LeafFromBool(b))
}

func (w *Wrapper) PutBytes(b []byte) {
	w.AddBytes(b)
}

func (w *Wrapper) PutUint16(i uint16) {
	w.AddUint16(i)
}

func (w *Wrapper) PutUint64(i uint64) {
	w.AddUint64(i)
}

func (w *Wrapper) PutUint8(i uint8) {
	w.AddUint8(i)
}

func (w *Wrapper) PutUint32(i uint32) {
	w.AddUint32(i)
}

/// --- legacy ones ---

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}

func (w *Wrapper) AddBytes(b []byte) {
	if len(b) <= 32 {
		w.AddNode(LeafFromBytes(b))
	} else {
		indx := w.Index()
		w.appendBytesAsNodes(b)
		w.Commit(indx)
	}
}

func (w *Wrapper) AddUint64(i uint64) {
	w.AddNode(LeafFromUint64(i))
}

func (w *Wrapper) AddUint32(i uint32) {
	w.AddNode(LeafFromUint32(i))
}

func (w *Wrapper) AddUint16(i uint16) {
	w.AddNode(LeafFromUint16(i))
}

func (w *Wrapper) AddUint8(i uint8) {
	w.AddNode(LeafFromUint8(i))
}

func (w *Wrapper) AddNode(n *Node) {
	if w.nodes == nil {
		w.nodes = []*Node{}
	}
	w.nodes = append(w.nodes, n)
}

func (w *Wrapper) Node() *Node {
	if len(w.nodes) != 1 {
		panic("BAD")
	}
	return w.nodes[0]
}

func (w *Wrapper) Hash() []byte {
	return w.nodes[len(w.nodes)-1].Hash()
}

func (w *Wrapper) Commit(i int) {
	w.fillEmptyNodes(i)

	res, err := TreeFromNodes(w.nodes[i:])
	if err != nil {
		panic(err)
	}
	// remove the old nodes
	w.nodes = w.nodes[:i]
	// add the new node
	w.AddNode(res)
}

func (w *Wrapper) CommitWithMixin(i, num, limit int) {
	w.fillEmptyNodes(i)
	res, err := TreeFromNodesWithMixin(w.nodes[i:], num, limit)
	if err != nil {
		panic(err)
	}
	// remove the old nodes
	w.nodes = w.nodes[:i]
	// add the new node
	w.AddNode(res)
}

func (w *Wrapper) AddEmpty() {
	w.AddNode(EmptyLeaf())
}

func (w *Wrapper) fillEmptyNodes(i int) {
	size := len(w.nodes[i:])
	for i := size; i < int(nextPowerOfTwo(uint64(size))); i++ {
		w.nodes = append(w.nodes, EmptyLeaf())
	}
}

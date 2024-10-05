package entrydb

// Frame is a single chunk of data
type Frame struct {
	Type  byte
	Index byte
	Total byte
	Data  []byte
}

func (f Frame) Encode() []byte {
	ret := make([]byte, 3+len(f.Data))
	ret[0] = f.Type
	ret[1] = f.Index
	ret[2] = f.Total
	copy(ret[3:], f.Data)
	return ret
}

// Overhead returns the number of bytes used by the frame
// it is always 3 bytes for the type, index, and total fields
func (f Frame) Overhead() int {
	return 3
}

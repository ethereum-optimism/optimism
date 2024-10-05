package entrydb

type EntryHeader struct {
	Type       byte
	FrameCount byte
	ID         SequenceValue
}

func (e EntryHeader) Frame() Frame {
	return Frame{
		Type:  e.Type,
		Index: 0,
		Total: e.FrameCount,
		Data:  e.ID[:],
	}
}

type Entry interface {
	Type() byte
	Encode() (EntryHeader, []byte, error)
	Decode(EntryHeader, []byte) (Entry, error)
	SequenceValue() SequenceValue
}

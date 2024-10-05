package entrydb

import (
	"errors"
	"io"
)

/* Entry DB
EntryDB has the following constraints:
- Frame based data chunking
- Regular interval checkpoints
- Append only
- Non-Decrementing sequence numbers only
- Fixed-size byte identifier for each entry
- Maximum 255 data types (0-254, 255 is reserved for checkpoints)
- Maximum 256 Frames per entry

From these constraints we get the following properties:
- O(1) append time
- O(log n) lookup time using binary search over checkpoints
- Configurable data width for compact use cases
- Corruption resistant using checkpoint trimming
*/

const SequenceValueSize = 32
const CheckpointType = byte(255)

var ErrSequenceValueDecrease = errors.New("sequence values must not decrease")

type SequenceValue [SequenceValueSize]byte

type EntryDB interface {
	Get(sv SequenceValue, e Entry) (Entry, error)
	Put(entry Entry) error
}

type entryDB struct {
	typeMap        map[byte]Entry
	width          int // true Frame width is width + 2
	length         int
	checkpointFreq int
	persistence    ReaderWriterSeeker

	// track the last written sequence value
	lastWritten SequenceValue
}

func NewEntryDB(
	typeMap map[byte]Entry,
	width int,
	checkpointFreq int,
	persistence ReaderWriterSeeker) EntryDB {
	if _, ok := typeMap[CheckpointType]; ok {
		panic("typeMap cannot overload CheckpointType")
	}
	if width < SequenceValueSize {
		panic("width must be at least SequenceValueSize")
	}
	return &entryDB{
		typeMap:        typeMap,
		width:          width,
		checkpointFreq: checkpointFreq,
		persistence:    persistence,
	}
}

func (db *entryDB) Put(entry Entry) error {
	db.persistence.Seek(0, io.SeekEnd)
	header, data, err := entry.Encode()
	if err != nil {
		return err
	}
	if db.lastWritten != (SequenceValue{}) {
		if compareSequenceValues(db.lastWritten, header.ID) == 1 {
			return ErrSequenceValueDecrease
		}
	}
	// store the header as the first frame
	frames := []Frame{header.Frame()}
	// turn the data into frames with width-sized data
	for i := 0; i < len(data); i += db.width {
		j := i + db.width
		if j > len(data) {
			j = len(data)
		}
		// offset is 1-indexed because the first frame is the header
		offset := (i / db.width) + 1
		frame := Frame{
			Type:  header.Type,
			Index: byte(offset),
			Data:  data[i:j],
		}
		frames = append(frames, frame)
	}
	// attach the total frame count to each frame
	for i := range frames {
		frames[i].Total = byte(len(frames))
	}
	// write all the frames
	for i := 0; i < len(frames); i++ {
		err := db.writeFrame(frames[i])
		if err != nil {
			return err
		}
		// if this is the last frame, update the last written sequence value
		// prior to checkpointing
		if i == len(frames)-1 {
			db.lastWritten = header.ID
		}
		err = db.MaybeCheckpoint()
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *entryDB) frameSize() int {
	return db.width + Frame{}.Overhead()
}

func (db *entryDB) writeFrame(frame Frame) error {
	fbytes := make([]byte, db.frameSize())
	copy(fbytes, frame.Encode())
	_, err := db.persistence.Write(fbytes)
	if err != nil {
		return err
	}
	db.length++
	return nil
}

func (db *entryDB) readFrame() Frame {
	buf := make([]byte, db.width+Frame{}.Overhead())
	db.persistence.Read(buf)
	return Frame{
		Type:  buf[0],
		Index: buf[1],
		Total: buf[2],
		Data:  buf[3:],
	}
}

func (db *entryDB) readFrames(n byte) []Frame {
	frames := make([]Frame, n)
	for i := 0; i < int(n); i++ {
		frames[i] = db.readFrame()
	}
	return frames
}

func framesToBytes(frames []Frame) []byte {
	ret := []byte{}
	for _, frame := range frames {
		ret = append(ret, frame.Data...)
	}
	return ret
}

func (db *entryDB) MaybeCheckpoint() error {
	if db.length%db.checkpointFreq != 0 {
		return nil
	}
	if db.lastWritten == (SequenceValue{}) {
		// this could happen if the first entry written creates more frames
		// should be impossible for checkpointFreq > 256
		panic("lastWritten should not be empty")
	}
	checkpoint := Frame{
		Type:  CheckpointType,
		Index: 0,
		Total: 1,
		Data:  db.lastWritten[:],
	}
	err := db.writeFrame(checkpoint)
	if err != nil {
		return err
	}
	return nil
}

func (db *entryDB) Get(sequenceValue SequenceValue, e Entry) (Entry, error) {
	if db.length == 0 {
		return nil, errors.New("database is empty")
	}
	err := db.seekToCheckpointPriorTo(sequenceValue)
	if err != nil {
		return nil, err
	}
	for {
		headerFrame := db.readFrame()
		if headerFrame.Index != 0 {
			panic("expected entry header")
		}
		sv := SequenceValue(headerFrame.Data[:SequenceValueSize])
		if compareSequenceValues(sv, sequenceValue) == 0 {
			h := EntryHeader{
				Type:       headerFrame.Type,
				FrameCount: headerFrame.Total,
				ID:         sv,
			}
			// we already read the header, so we need to read the rest of the frames
			fs := db.readFrames(headerFrame.Total - 1)
			return e.Decode(h, framesToBytes(fs))
		}
		if compareSequenceValues(sv, sequenceValue) == -1 {
			return nil, errors.New("sequence value not found")
		}
		db.seekFrames(int(headerFrame.Total) - 1)
	}
}

// seekFrames advances the persistence by the width of a frame
func (db *entryDB) seekFrames(n int) error {
	size := db.width + Frame{}.Overhead()
	return db.seekAdvance(n * size)
}

func (db *entryDB) seekAdvance(n int) error {
	_, err := db.persistence.Seek(int64(n), io.SeekCurrent)
	return err
}

// seekToCheckpointPriorTo travels to the index of the first checkpoint behind the given sequence value
// it can be used to seek to a known good state before the given sequence value
// TODO: this function can be a binary search instead of a linear search
func (db *entryDB) seekToCheckpointPriorTo(sequenceValue SequenceValue) error {
	if db.length == 0 {
		return errors.New("database is empty")
	}
	if compareSequenceValues(sequenceValue, db.lastWritten) == 1 {
		return errors.New("sequence value is ahead of last written")
	}
	// start at the beginning
	db.persistence.Seek(0, io.SeekStart)
	// if there are fewer frames than the checkpoint frequency, we can't seek
	if db.length <= db.checkpointFreq {
		return nil
	}
	db.seekFrames(db.checkpointFreq)
	f := 0
	for {
		db.seekFrames(db.checkpointFreq)
		f += db.checkpointFreq
		frame := db.readFrame()
		if frame.Type != CheckpointType {
			panic("expected checkpoint type")
		}
		sv := SequenceValue(frame.Data[:SequenceValueSize])
		if compareSequenceValues(sv, sequenceValue) > 0 {
			break
		}
		// reverse one frame to get back to the checkpoint
		db.seekFrames(-1)
	}
	db.seekFrames(-1)
	db.seekFrames(-1 * db.checkpointFreq)
	// once we exit the loop, rewind one checkpoint
	return nil
}

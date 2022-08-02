package derive

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Frames cannot be larger than 1 MB.
// Data transactions that carry frames are generally not larger than 128 KB due to L1 network conditions,
// but we leave space to grow larger anyway (gas limit allows for more data).
const MaxFrameLen = 1_000_000

var ErrNotEnoughFrameBytes = errors.New("not enough available bytes for the frame")

// Data Format
//
// frame = channel_id ++ frame_number ++ frame_data_length ++ frame_data ++ is_last
//
// channel_id        = random ++ timestamp
// random            = bytes32
// timestamp         = uvarint
// frame_number      = uvarint
// frame_data_length = uvarint
// frame_data        = bytes
// is_last           = bool

type Frame struct {
	ID          ChannelID
	FrameNumber uint64
	Data        []byte
	IsLast      bool
}

// MarshalBinary writes the frame to `w`.
// It returns the number of bytes written as well as any
// error encountered while writing.
func (f *Frame) MarshalBinary(w io.Writer) (int, error) {
	n, err := w.Write(f.ID.Data[:])
	if err != nil {
		return n, err
	}
	l, err := w.Write(makeUVarint(f.ID.Time))
	n += l
	if err != nil {
		return n, err
	}
	l, err = w.Write(makeUVarint(f.FrameNumber))
	n += l
	if err != nil {
		return n, err
	}

	l, err = w.Write(makeUVarint(uint64(len(f.Data))))
	n += l
	if err != nil {
		return n, err
	}
	l, err = w.Write(f.Data)
	n += l
	if err != nil {
		return n, err
	}
	if f.IsLast {
		l, err = w.Write([]byte{1})
		n += l
		if err != nil {
			return n, err
		}
	} else {
		l, err = w.Write([]byte{0})
		n += l
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

type ByteReader interface {
	io.Reader
	io.ByteReader
}

// UnmarshalBinary consumes a full frame from the reader.
// If `r` fails a read, it returns the error from the reader
// The reader will be left in a partially read state.
func (f *Frame) UnmarshalBinary(r ByteReader) error {
	_, err := io.ReadFull(r, f.ID.Data[:])
	if err != nil {
		return fmt.Errorf("error reading ID: %w", err)
	}
	f.ID.Time, err = binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("error reading ID.Time: %w", err)
	}
	// stop reading and ignore remaining data if we encounter a zeroed ID
	if f.ID == (ChannelID{}) {
		return io.EOF
	}
	f.FrameNumber, err = binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("error reading frame number: %w", err)
	}

	frameLength, err := binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("error reading frame length: %w", err)
	}

	// Cap frame length to MaxFrameLen (currently 1MB)
	if frameLength > MaxFrameLen {
		return fmt.Errorf("frameLength is too large: %d", frameLength)
	}
	f.Data = make([]byte, int(frameLength))
	if _, err := io.ReadFull(r, f.Data); err != nil {
		return fmt.Errorf("error reading frame data: %w", err)
	}

	isLastByte, err := r.ReadByte()
	if err != nil && err != io.EOF {
		return fmt.Errorf("error reading final byte: %w", err)
	}
	if isLastByte == 0 {
		f.IsLast = false
	} else if isLastByte == 1 {
		f.IsLast = true
	} else {
		return errors.New("invalid byte as is_last")
	}
	return err
}

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

// Data Format
//
// frame = channel_id ++ frame_number ++ frame_data_length ++ frame_data ++ is_last
//
// channel_id        = random ++ timestamp
// random            = bytes32
// timestamp         = uint64
// frame_number      = uint16
// frame_data_length = uint32
// frame_data        = bytes
// is_last           = bool

type Frame struct {
	ID          ChannelID
	FrameNumber uint16
	Data        []byte
	IsLast      bool
}

// MarshalBinary writes the frame to `w`.
// It returns any errors encountered while writing, but
// generally expects the writer very rarely fail.
func (f *Frame) MarshalBinary(w io.Writer) error {
	_, err := w.Write(f.ID.Data[:])
	if err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, f.ID.Time); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, f.FrameNumber); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint32(len(f.Data))); err != nil {
		return err
	}
	_, err = w.Write(f.Data)
	if err != nil {
		return err
	}
	if f.IsLast {
		if _, err = w.Write([]byte{1}); err != nil {
			return err
		}
	} else {
		if _, err = w.Write([]byte{0}); err != nil {
			return err
		}
	}
	return nil
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
	if err := binary.Read(r, binary.BigEndian, &f.ID.Time); err != nil {
		return fmt.Errorf("error reading ID time: %w", err)
	}
	// stop reading and ignore remaining data if we encounter a zeroed ID
	if f.ID == (ChannelID{}) {
		return io.EOF
	}

	if err := binary.Read(r, binary.BigEndian, &f.FrameNumber); err != nil {
		return fmt.Errorf("error reading frame number: %w", err)
	}

	var frameLength uint32
	if err := binary.Read(r, binary.BigEndian, &frameLength); err != nil {
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

	if isLastByte, err := r.ReadByte(); err != nil && err != io.EOF {
		return fmt.Errorf("error reading final byte: %w", err)
	} else if isLastByte == 0 {
		f.IsLast = false
	} else if isLastByte == 1 {
		f.IsLast = true
	} else {
		return errors.New("invalid byte as is_last")
	}
	return err
}

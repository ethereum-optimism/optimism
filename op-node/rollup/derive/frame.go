package derive

import (
	"bytes"
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
// channel_id        = bytes16
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
	_, err := w.Write(f.ID[:])
	if err != nil {
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
	if _, err := io.ReadFull(r, f.ID[:]); err != nil {
		return fmt.Errorf("error reading ID: %w", err)
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
		return err
	} else if isLastByte == 1 {
		f.IsLast = true
		return err
	} else {
		return errors.New("invalid byte as is_last")
	}
}

// Frames on stored in L1 transactions with the following format:
// data = DerivationVersion0 ++ Frame(s)
// Where there is one or more frames concatenated together.

// ParseFrames parse the on chain serialization of frame(s) in
// an L1 transaction. Currently only version 0 of the serialization
// format is supported.
// All frames must be parsed without error and there must not be
// any left over data and there must be at least one frame.
func ParseFrames(data []byte) ([]Frame, error) {
	if len(data) == 0 {
		return nil, errors.New("data array must not be empty")
	}
	if data[0] != DerivationVersion0 {
		return nil, fmt.Errorf("invalid derivation format byte: got %d", data[0])
	}
	buf := bytes.NewBuffer(data[1:])
	var frames []Frame
	for buf.Len() > 0 {
		var f Frame
		if err := (&f).UnmarshalBinary(buf); err != io.EOF && err != nil {
			return nil, err
		}
		frames = append(frames, f)
	}
	if buf.Len() != 0 {
		return nil, fmt.Errorf("did not fully consume data: have %d frames and %d bytes left", len(frames), buf.Len())
	}
	if len(frames) == 0 {
		return nil, errors.New("was not able to find any frames")
	}
	return frames, nil
}

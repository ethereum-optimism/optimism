package preimage

import (
	"encoding/binary"
	"fmt"
	"io"
)

// HintWriter writes hints to an io.Writer (e.g. a special file descriptor, or a debug log),
// for a pre-image oracle service to prepare specific pre-images.
type HintWriter struct {
	rw io.ReadWriter
}

var _ Hinter = (*HintWriter)(nil)

func NewHintWriter(rw io.ReadWriter) *HintWriter {
	return &HintWriter{rw: rw}
}

func (hw *HintWriter) Hint(v Hint) {
	hint := v.Hint()
	var hintBytes []byte
	hintBytes = binary.BigEndian.AppendUint32(hintBytes, uint32(len(hint)))
	hintBytes = append(hintBytes, []byte(hint)...)
	_, err := hw.rw.Write(hintBytes)
	if err != nil {
		panic(fmt.Errorf("failed to write pre-image hint: %w", err))
	}
	_, err = hw.rw.Read([]byte{0})
	if err != nil {
		panic(fmt.Errorf("failed to read pre-image hint ack: %w", err))
	}
}

// HintReader reads the hints of HintWriter and passes them to a router for preparation of the requested pre-images.
// Onchain the written hints are no-op.
type HintReader struct {
	rw io.ReadWriter
}

func NewHintReader(rw io.ReadWriter) *HintReader {
	return &HintReader{rw: rw}
}

type HintHandler func(hint string) error

func (hr *HintReader) NextHint(router HintHandler) error {
	var length uint32
	if err := binary.Read(hr.rw, binary.BigEndian, &length); err != nil {
		if err == io.EOF {
			return io.EOF
		}
		return fmt.Errorf("failed to read hint length prefix: %w", err)
	}
	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(hr.rw, payload); err != nil {
			return fmt.Errorf("failed to read hint payload (length %d): %w", length, err)
		}
	}
	if err := router(string(payload)); err != nil {
		// write back on error to unblock the HintWriter
		_, _ = hr.rw.Write([]byte{0})
		return fmt.Errorf("failed to handle hint: %w", err)
	}
	if _, err := hr.rw.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to write trailing no-op byte to unblock hint writer: %w", err)
	}
	return nil
}

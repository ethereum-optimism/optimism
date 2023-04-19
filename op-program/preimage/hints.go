package preimage

import (
	"encoding/binary"
	"fmt"
	"io"
)

// HintWriter writes hints to an io.Writer (e.g. a special file descriptor, or a debug log),
// for a pre-image oracle service to prepare specific pre-images.
type HintWriter struct {
	w io.Writer
}

var _ Hinter = (*HintWriter)(nil)

func NewHintWriter(w io.Writer) *HintWriter {
	return &HintWriter{w: w}
}

func (hw *HintWriter) Hint(v Hint) {
	hint := v.Hint()
	var hintBytes []byte
	hintBytes = binary.BigEndian.AppendUint32(hintBytes, uint32(len(hint)))
	hintBytes = append(hintBytes, []byte(hint)...)
	hintBytes = append(hintBytes, 0) // to block writing on
	_, err := hw.w.Write(hintBytes)
	if err != nil {
		panic(fmt.Errorf("failed to write pre-image hint: %w", err))
	}
}

// HintReader reads the hints of HintWriter and passes them to a router for preparation of the requested pre-images.
// Onchain the written hints are no-op.
type HintReader struct {
	r io.Reader
}

func NewHintReader(r io.Reader) *HintReader {
	return &HintReader{r: r}
}

func (hr *HintReader) NextHint(router func(hint string) error) error {
	var length uint32
	if err := binary.Read(hr.r, binary.BigEndian, &length); err != nil {
		if err == io.EOF {
			return io.EOF
		}
		return fmt.Errorf("failed to read hint length prefix: %w", err)
	}
	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(hr.r, payload); err != nil {
			return fmt.Errorf("failed to read hint payload (length %d): %w", length, err)
		}
	}
	if err := router(string(payload)); err != nil {
		// stream recovery
		_, _ = hr.r.Read([]byte{0})
		return fmt.Errorf("failed to handle hint: %w", err)
	}
	if _, err := hr.r.Read([]byte{0}); err != nil {
		return fmt.Errorf("failed to read trailing no-op byte to unblock hint writer: %w", err)
	}
	return nil
}

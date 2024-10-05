package entrydb

import "io"

// Wrap the io.Reader, io.Writer, and io.Seeker interfaces into a single interface
type ReaderWriterSeeker interface {
	io.Reader
	io.Writer
	io.Seeker
}

// CompareBytes compares two SequenceValues
// which are just fixed-size byte arrays
// it returns -1 if a < b, 0 if a == b, and 1 if a > b
func compareSequenceValues(a, b SequenceValue) int {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return -1 // a is less than b
		} else if a[i] > b[i] {
			return 1 // a is greater than b
		}
	}
	return 0 // a is equal to b
}

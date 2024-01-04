package eth

import (
	"testing"
)

func TestBlobEncodeDecode(t *testing.T) {
	cases := []string{
		"this is a test of blob encoding/decoding",
		"short",
		"\x00",
		"\x00\x01\x00",
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
		"",
	}

	var b Blob
	for _, c := range cases {
		data := Data(c)
		if err := b.FromData(data); err != nil {
			t.Fatalf("failed to encode bytes: %v", err)
		}
		decoded, err := b.ToData()
		if err != nil {
			t.Fatalf("failed to decode blob: %v", err)
		}
		if string(decoded) != c {
			t.Errorf("decoded != input. got: %v, want: %v", decoded, Data(c))
		}
	}
}

func TestBigBlobEncoding(t *testing.T) {
	bigData := Data(make([]byte, MaxBlobDataSize))
	bigData[MaxBlobDataSize-1] = 0xFF
	var b Blob
	if err := b.FromData(bigData); err != nil {
		t.Fatalf("failed to encode bytes: %v", err)
	}
	decoded, err := b.ToData()
	if err != nil {
		t.Fatalf("failed to decode blob: %v", err)
	}
	if string(decoded) != string(bigData) {
		t.Errorf("decoded blob != big blob input")
	}
}

func TestInvalidBlobDecoding(t *testing.T) {
	data := Data("this is a test of invalid blob decoding")
	var b Blob
	if err := b.FromData(data); err != nil {
		t.Fatalf("failed to encode bytes: %v", err)
	}
	b[32] = 0x80 // field elements should never have their highest order bit set
	if _, err := b.ToData(); err == nil {
		t.Errorf("expected error, got none")
	}

	b[32] = 0x00
	b[4] = 0xFF // encode an invalid (much too long) length prefix
	if _, err := b.ToData(); err == nil {
		t.Errorf("expected error, got none")
	}
}

func TestTooLongDataEncoding(t *testing.T) {
	// should never be able to encode data that has size the same as that of the blob due to < 256
	// bit precision of each field element
	data := Data(make([]byte, BlobSize))
	var b Blob
	err := b.FromData(data)
	if err == nil {
		t.Errorf("expected error, got none")
	}
}

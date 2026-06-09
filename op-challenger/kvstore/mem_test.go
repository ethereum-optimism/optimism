package kvstore

import "testing"

func TestMemKV(t *testing.T) {
	kv := NewMemKV()
	kvTest(t, kv)
}

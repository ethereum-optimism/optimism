package kvstore

import "testing"

func TestDiskKV(t *testing.T) {
	tmp := t.TempDir() // automatically removed by testing cleanup
	kv := NewDiskKV(tmp)
	kvTest(t, kv)
}

package diffdb

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDiffDb(t *testing.T) {
	db, err := NewDiffDb("./test_diff.db", 3)
	// cleanup (sqlite will create the file if it doesn't exist)
	defer os.Remove("./test_diff.db")
	if err != nil {
		t.Fatal(err)
	}

	hashes := []common.Hash{
		common.Hash{0x0},
		common.Hash{0x1},
		common.Hash{0x2},
	}
	addr := common.Address{0x1}
	db.SetDiffKey(big.NewInt(1), common.Address{0x1, 0x2}, common.Hash{0x12, 0x13}, false)
	db.SetDiffKey(big.NewInt(1), addr, hashes[0], false)
	db.SetDiffKey(big.NewInt(1), addr, hashes[1], false)
	db.SetDiffKey(big.NewInt(1), addr, hashes[2], false)
	db.SetDiffKey(big.NewInt(1), common.Address{0x2}, common.Hash{0x99}, false)
	db.SetDiffKey(big.NewInt(2), common.Address{0x2}, common.Hash{0x98}, true)
	// try overwriting, ON CONFLICT clause gets hit
	err = db.SetDiffKey(big.NewInt(2), common.Address{0x2}, common.Hash{0x98}, false)
	if err != nil {
		t.Fatal("should be able to resolve conflict without error at the sql level")
	}

	diff, err := db.GetDiff(big.NewInt(1))
	if err != nil {
		t.Fatal("Did not expect error")
	}
	for i := range hashes {
		if hashes[i] != diff[addr][i].Key {
			t.Fatal("Did not match", hashes[i], "got", diff[addr][i].Key)
		}
	}

	diff, _ = db.GetDiff(big.NewInt(2))
	if diff[common.Address{0x2}][0].Mutated != true {
		t.Fatalf("Did not match mutated")
	}
}

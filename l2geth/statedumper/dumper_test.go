package statedumper

import (
	"github.com/ethereum-optimism/optimism/l2geth/common"
	"io"
	"os"
	"testing"
)

func TestFileStateDumper(t *testing.T) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("error creating file: %v", err)
	}
	err = os.Setenv("L2GETH_STATE_DUMP_PATH", f.Name())
	if err != nil {
		t.Fatalf("error setting env file: %v", err)
	}
	dumper := NewStateDumper()
	addr := common.Address{19: 0x01}
	dumper.WriteETH(addr)
	dumper.WriteMessage(addr, []byte("hi"))
	_, err = f.Seek(0, 0)
	if err != nil {
		t.Fatalf("error seeking: %v", err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("error reading: %v", err)
	}
	dataStr := string(data)
	if dataStr != "ETH|0x0000000000000000000000000000000000000001\nMSG|0x0000000000000000000000000000000000000001|6869\n" {
		t.Fatalf("invalid data. got: %s", dataStr)
	}
}

package sources

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestRethDBRead(t *testing.T) {
	t.Parallel()

	_, err := FetchRethReceipts("/test", &common.Hash{})
	if err != nil {
		panic("test")
	}
}

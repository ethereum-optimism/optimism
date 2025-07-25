package metrics

import (
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestNoopMetrics(t *testing.T) {
	m := &NoopMetrics{}
	m.RecordInfo("1234")
	m.RecordUp()
	onDone := m.RecordFundAction("faucetA", eth.ChainIDFromUInt64(123), eth.OneEther)
	onDone(errors.New("test err"))
	m.RecordNonce(123)
}

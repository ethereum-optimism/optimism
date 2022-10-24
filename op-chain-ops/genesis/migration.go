package genesis

import (
	"encoding/json"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// SentMessageJSON represents an entry in the JSON file that is created by
// the `migration-data` package. Each entry represents a call to the
// `LegacyMessagePasser`. The `who` should always be the
// `L2CrossDomainMessenger` and the `msg` should be an abi encoded
// `relayMessage(address,address,bytes,uint256)`
type SentMessageJSON struct {
	Who common.Address `json:"who"`
	Msg hexutil.Bytes  `json:"msg"`
}

// NewSentMessageJSON will read a JSON file from disk given a path to the JSON
// file. The JSON file this function reads from disk is an output from the
// `migration-data` package.
func NewSentMessageJSON(path string) ([]*SentMessageJSON, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var j []*SentMessageJSON
	if err := json.Unmarshal(file, &j); err != nil {
		return nil, err
	}

	return j, nil
}

// ToLegacyWithdrawal will convert a SentMessageJSON to a LegacyWithdrawal
// struct. This is useful because the LegacyWithdrawal struct has helper
// functions on it that can compute the withdrawal hash and the storage slot.
func (s *SentMessageJSON) ToLegacyWithdrawal() (*crossdomain.LegacyWithdrawal, error) {
	data := make([]byte, 0, len(s.Who)+len(s.Msg))
	copy(data, s.Msg)
	copy(data[len(s.Msg):], s.Who[:])

	var w crossdomain.LegacyWithdrawal
	if err := w.Decode(data); err != nil {
		return nil, err
	}
	return &w, nil
}

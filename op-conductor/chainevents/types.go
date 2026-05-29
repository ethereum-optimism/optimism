package chainevents

import "encoding/json"

// canonStateNotification mirrors reth's externally-tagged CanonStateNotification.
// Only the variant discriminator and each chain's block numbers are decoded; the
// heavy execution_outcome / trie_data / block bodies in the frame are ignored.
//
// The reth notification does NOT carry block hashes (reth's SealedHeader.hash is
// serde-skipped), so we extract only block numbers here. The reorg handler reads
// the authoritative head hash from the EL via InfoByLabel(eth.Unsafe).
type canonStateNotification struct {
	Commit *commitNotification `json:"Commit,omitempty"`
	Reorg  *reorgNotification  `json:"Reorg,omitempty"`
}

type commitNotification struct {
	New chain `json:"new"`
}

type reorgNotification struct {
	Old chain `json:"old"`
	New chain `json:"new"`
}

// chain mirrors reth's Chain, decoding only the blocks map. reth serializes the
// block-number keys as decimal strings, which encoding/json parses into uint64.
type chain struct {
	Blocks map[uint64]json.RawMessage `json:"blocks"`
}

// tipNumber returns the highest block number in the chain and whether the chain
// is non-empty. reth's "new" chain can be empty on a pure revert.
func (c *chain) tipNumber() (uint64, bool) {
	var tip uint64
	found := false
	for n := range c.Blocks {
		if !found || n > tip {
			tip = n
			found = true
		}
	}
	return tip, found
}

// subscribeResponse is the JSON-RPC reply to the subscribe request. A successful
// response carries the subscription id as a hex string in Result.
type subscribeResponse struct {
	ID     int              `json:"id"`
	Result string           `json:"result"`
	Error  *json.RawMessage `json:"error,omitempty"`
}

// notificationEnvelope is the JSON-RPC notification pushed for each chain event:
// {"method":"reth_subscribeChainNotifications","params":{"subscription":"0x..","result":<CanonStateNotification>}}
type notificationEnvelope struct {
	Method string `json:"method"`
	Params struct {
		Subscription string          `json:"subscription"`
		Result       json.RawMessage `json:"result"`
	} `json:"params"`
}

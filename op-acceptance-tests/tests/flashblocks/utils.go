package flashblocks

import (
	"encoding/json"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/stretchr/testify/require"
)

type Flashblock struct {
	PayloadID string `json:"payload_id"`
	Index     int    `json:"index"`
	Diff      struct {
		StateRoot    string `json:"state_root"`
		ReceiptsRoot string `json:"receipts_root"`
		LogsBloom    string `json:"logs_bloom"`
		GasUsed      string `json:"gas_used"`
		BlockHash    string `json:"block_hash"`
		Transactions []any  `json:"transactions"`
		Withdrawals  []any  `json:"withdrawals"`
	} `json:"diff"`
	Metadata struct {
		BlockNumber        int                    `json:"block_number"`
		NewAccountBalances map[string]string      `json:"new_account_balances"`
		Receipts           map[string]interface{} `json:"receipts"`
	} `json:"metadata"`
}

type FlashblocksStreamMode string

const (
	FlashblocksStreamMode_Leader   FlashblocksStreamMode = "leader"
	FlashblocksStreamMode_Follower FlashblocksStreamMode = "follower"
)

// UnmarshalJSON implements custom unmarshaling for Flashblock to lower case the keys of .metadata.new_account_balances.
func (f *Flashblock) UnmarshalJSON(data []byte) error {
	type TempFlashblock Flashblock // need a type alias to avoid infinite recursion
	temp := (*TempFlashblock)(f)

	if err := json.Unmarshal(data, temp); err != nil {
		return err
	}
	if f.Metadata.NewAccountBalances == nil {
		return nil
	}

	loweredBalances := make(map[string]string)
	for key, value := range f.Metadata.NewAccountBalances {
		loweredBalances[strings.ToLower(key)] = value
	}
	f.Metadata.NewAccountBalances = loweredBalances

	return nil
}

// DriveViaTestSequencer explicitly builds a few blocks to ensure the builder/rollup-boost
// have payloads to serve before we start listening for flashblocks.
func DriveViaTestSequencer(t devtest.T, sys *presets.SingleChainWithFlashblocks, count int) {
	t.Helper()
	ts := sys.TestSequencer.Escape().ControlAPI(sys.L2Chain.ChainID())
	ctx := t.Ctx()

	head := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	for range count {
		require.NoError(t, ts.New(ctx, seqtypes.BuildOpts{Parent: head.Hash}))
		require.NoError(t, ts.Next(ctx))
		head = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	}
	// Ensure the sequencer EL has produced at least one unsafe block before subscribing.
	sys.L2EL.WaitForBlockNumber(1)

	// Log the latest unsafe head and L1 origin to confirm block production before listening.
	head = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	sys.Log.Info("Pre-listen unsafe head", "unsafe", head)
}

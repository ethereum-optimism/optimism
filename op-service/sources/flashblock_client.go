package sources

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/log"
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

type WebsocketReader interface {
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
}

// FlashblockClient wraps a WSClient and delivers parsed Flashblock values on a channel.
type FlashblockClient struct {
	ws     WebsocketReader
	ch     chan *Flashblock
	logger log.Logger
}

// NewFlashblockClient creates a new FlashblockClient. Call Start to begin reading.
func NewFlashblockClient(ws WebsocketReader, logger log.Logger, bufferSize uint) *FlashblockClient {
	return &FlashblockClient{
		ws:     ws,
		ch:     make(chan *Flashblock, bufferSize),
		logger: logger,
	}
}

// Start begins the background read loop. It reads from the websocket until ctx
// is cancelled, unmarshals each message into a *Flashblock, and sends it on the
// channel returned by Next. When the loop exits, a nil is sent and the channel
// is closed.
func (c *FlashblockClient) Start(ctx context.Context) error {
	defer close(c.ch)

	for {
		_, msg, err := c.ws.Read(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || ctx.Err() != nil {
				c.logger.Info("FlashblockClient: read loop finished")
				return nil
			}
			c.logger.Error("FlashblockClient: read error", "err", err)
			return err
		}

		fb := new(Flashblock)
		if err := json.Unmarshal(msg, fb); err != nil {
			c.logger.Warn("FlashblockClient: unmarshal error, skipping", "err", err)
			continue
		}

		select {
		case c.ch <- fb:
		default:
			c.logger.Warn("FlashblockClient: channel full, dropping flashblock",
				"block_number", fb.Metadata.BlockNumber, "index", fb.Index)
		}
	}
}

// Next returns the receive-only channel of parsed flashblocks.
// A nil value signals that the read loop has exited; the channel is then closed.
func (c *FlashblockClient) Next() <-chan *Flashblock {
	return c.ch
}

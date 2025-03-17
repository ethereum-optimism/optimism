package txplan

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/plan"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type InitTrigger struct {
	Emitter    common.Address // address of the EventLogger contract
	Topics     []common.Hash
	OpaqueData []byte
}

func (v *InitTrigger) To() (*common.Address, error) {
	return &v.Emitter, nil
}

func (v *InitTrigger) Data() ([]byte, error) {
	// TODO format call
	return nil, nil
}

type ExecTrigger struct {
	Executor common.Address // address of the EventLogger contract
	Msg      suptypes.Message
}

func (v *ExecTrigger) To() (*common.Address, error) {
	return &v.Executor, nil
}

func (v *ExecTrigger) Data() ([]byte, error) {
	// TODO format call to CrossL2Inbox
	return nil, nil
}

type Call interface {
	To() (*common.Address, error)
	Data() ([]byte, error)
	// AccessList
}

type MultiTrigger struct {
	Calls []Call
}

func (v *MultiTrigger) Data() ([]byte, error) {
	// TODO format multi-call
	return nil, nil
}

type Result interface {
	FromReceipt(ctx context.Context, rec *types.Receipt, includedIn eth.BlockRef, chainID eth.ChainID) error
}

type InteropOutput struct {
	Entries []suptypes.Message
}

func (i *InteropOutput) FromReceipt(ctx context.Context, rec *types.Receipt, includedIn eth.BlockRef, chainID eth.ChainID) error {
	for _, logEvent := range rec.Logs {
		payload := suptypes.LogToMessagePayload(logEvent)
		id := suptypes.Identifier{
			Origin:      logEvent.Address,
			BlockNumber: logEvent.BlockNumber,
			LogIndex:    uint32(logEvent.Index),
			Timestamp:   includedIn.Time,
			ChainID:     chainID,
		}
		payloadHash := crypto.Keccak256Hash(payload)
		i.Entries = append(i.Entries, suptypes.Message{
			Identifier:  id,
			PayloadHash: payloadHash,
		})
	}
	return nil
}

type IntentTx[V Call, R Result] struct {
	PlannedTx *PlannedTx
	Content   plan.Lazy[V]
	Result    plan.Lazy[R]
}

func NewIntent[V Call, R Result](opts ...Option) *IntentTx[V, R] {
	v := &IntentTx[V, R]{
		PlannedTx: NewPlannedTx(opts...),
	}
	v.PlannedTx.To.DependOn(&v.Content)
	v.PlannedTx.To.Fn(func(ctx context.Context) (*common.Address, error) {
		return v.Content.Value().To()
	})
	v.PlannedTx.Data.DependOn(&v.Content)
	v.PlannedTx.Data.Fn(func(ctx context.Context) (hexutil.Bytes, error) {
		return v.Content.Value().Data()
	})
	// TODO add access-list relation

	v.Result.DependOn(&v.PlannedTx.Included, &v.PlannedTx.IncludedBlock, &v.PlannedTx.ChainID)
	v.Result.Fn(func(ctx context.Context) (R, error) {
		var r R
		err := r.FromReceipt(ctx, v.PlannedTx.Included.Value(), v.PlannedTx.IncludedBlock.Value(), v.PlannedTx.ChainID.Value())
		return r, err
	})
	return v
}

func executeIndexed(events *plan.Lazy[*InteropOutput], index int) func(ctx context.Context) (*ExecTrigger, error) {
	return func(ctx context.Context) (*ExecTrigger, error) {
		if x := len(events.Value().Entries); x <= index {
			return nil, fmt.Errorf("invalid index: %d, only have %d events", index, x)
		}
		return &ExecTrigger{
			Executor: common.Address{},
			Msg:      events.Value().Entries[index],
		}, nil
	}
}

func TestInteropTx(t *testing.T) {
	t.Skip() // TODO

	eventLogger := common.Address{} // TODO deploy tx

	priv, err := crypto.GenerateKey()
	require.NoError(t, err)

	cl, err := sources.NewEthClient()
	require.NoError(t, err)

	opts := Combine(
		WithPrivateKey(priv),
		WithTransactionSubmitter(cl),
		WithAssumedInclusion(cl),
		WithRetryInclusion(10, retry.Exponential()),
		WithBlockInclusionInfo(cl),
	)

	txSimple := NewPlannedTx(opts, WithEth(big.NewInt(1234)))
	rec, err := txSimple.Included.Eval(context.Background())
	require.NoError(t, err)

	txA := NewIntent[*InitTrigger, *InteropOutput](opts)
	txA.Content.Set(&InitTrigger{
		Emitter:    eventLogger,
		Topics:     []common.Hash{},
		OpaqueData: []byte("hello world!"),
	})

	txB := NewIntent[*ExecTrigger, *InteropOutput]()
	txB.Content.DependOn(&txA.Result)
	txB.Content.Fn(executeIndexed(&txA.Result, 0))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	recA, err := txA.PlannedTx.Included.Eval(ctx)
	require.NoError(t, err)
	t.Logf("included initiating tx in block %s", recA.BlockHash)

	recB, err := txB.PlannedTx.Included.Eval(ctx)
	require.NoError(t, err)
	t.Logf("included executing tx in block %s", recB.BlockHash)
}

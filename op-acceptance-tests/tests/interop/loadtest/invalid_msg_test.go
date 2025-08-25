package loadtest

import (
	"context"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

type makeInvalidInitMsgFn func(suptypes.Message) suptypes.Message

func makeInvalidBlockNumber(msg suptypes.Message) suptypes.Message {
	msg.Identifier.BlockNumber++
	return msg
}

func makeInvalidChainID(msg suptypes.Message) suptypes.Message {
	chainIDBig := msg.Identifier.ChainID.ToBig()
	msg.Identifier.ChainID = eth.ChainIDFromBig(chainIDBig.Add(chainIDBig, big.NewInt(1)))
	return msg
}

func makeInvalidLogIndex(msg suptypes.Message) suptypes.Message {
	msg.Identifier.LogIndex++
	return msg
}

func makeInvalidOrigin(msg suptypes.Message) suptypes.Message {
	originBig := msg.Identifier.Origin.Big()
	msg.Identifier.Origin = common.BigToAddress(originBig.Add(originBig, big.NewInt(1)))
	return msg
}

func makeInvalidTimestamp(msg suptypes.Message) suptypes.Message {
	msg.Identifier.Timestamp++
	return msg
}

func makeInvalidPayloadHash(msg suptypes.Message) suptypes.Message {
	hash := msg.PayloadHash.Big()
	hash.Add(hash, big.NewInt(1))
	msg.PayloadHash = common.BigToHash(hash)
	return msg
}

// InvalidExecMsgSpammer spams invalid executing messages, aiming to stress mempool interop
// filters.
type InvalidExecMsgSpammer struct {
	l2             *L2
	eoa            *SyncEOA
	validInitMsg   suptypes.Message
	makeInvalidFns *RoundRobin[makeInvalidInitMsgFn]
}

var _ Spammer = (*InvalidExecMsgSpammer)(nil)

// NewInvalidExecMsgSpammer returns an InvalidExecutor. It assumes  validInitMsg is a valid
// initiating message on a source chain.
func NewInvalidExecMsgSpammer(t devtest.T, l2 *L2, validInitMsg suptypes.Message) *InvalidExecMsgSpammer {
	// Fund an EOA that will be spamming the invalid transactions. It should never need to spend
	// any wei, but we don't want to trigger mempool balance checks.
	eoa := l2.Wallet.NewEOA(l2.EL)
	address := eoa.Address()
	_, err := l2.Include(t, txplan.WithValue(eth.OneHundredthEther), txplan.WithTo(&address))
	t.Require().NoError(err)

	// The InvalidExecutor uses a txinclude.Includer to manage nonces concurrently. It uses a
	// txinclude.Sender implementation that fails on the first error. Because we anticipate
	// failure, this enables us to spam without entering infinite resubmission loops.
	unreliableSender := struct {
		*txinclude.Monitor
		txinclude.Sender
	}{
		Monitor: txinclude.NewMonitor(l2.EL.Escape().EthClient(), l2.BlockTime),
		Sender:  l2.EL.Escape().EthClient(),
	}
	signer := txinclude.NewPkSigner(eoa.Key().Priv(), l2.Config.ChainID)
	includer := txinclude.NewPersistent(signer, unreliableSender)

	return &InvalidExecMsgSpammer{
		l2:           l2,
		eoa:          NewSyncEOA(includer, eoa.Plan()),
		validInitMsg: validInitMsg,
		makeInvalidFns: NewRoundRobin([]makeInvalidInitMsgFn{
			makeInvalidBlockNumber,
			makeInvalidChainID,
			makeInvalidLogIndex,
			makeInvalidOrigin,
			makeInvalidTimestamp,
			makeInvalidPayloadHash,
		}),
	}
}

func (ie *InvalidExecMsgSpammer) Spam(t devtest.T) error {
	invalidInitMsg := ie.makeInvalidFns.Get()(ie.validInitMsg)
	execMsg := planExecMsg(t, &invalidInitMsg, ie.l2.BlockTime, ie.l2.EL.Escape().EthClient())
	if _, err := ie.eoa.Include(t, execMsg); err == nil {
		t.Require().Failf("included invalid executing message", "message: %v", invalidInitMsg)
	} else if !strings.Contains(err.Error(), core.ErrTxFilteredOut.Error()) { // TODO(13408): we should be able to use errors.Is.
		t.Logger().Warn("Invalid message hit an unexpected error", "want", core.ErrTxFilteredOut, "got", err)
	}
	return nil
}

// TestRelayWithInvalidMessagesSteady is equivalent to TestRelaySteady except that invalid
// executing messages are also spammed. The number of invalid messages spammed per slot is
// configurable via NAT_INVALID_MPS (default: 1_000).
func TestRelayWithInvalidMessagesSteady(gt *testing.T) {
	t, l2A, l2B := setupLoadTest(gt)

	// Emit a valid initiating message.
	initTx, err := l2A.Include(t, planCall(t, &txintent.InitTrigger{
		Emitter:    l2A.EventLogger,
		Topics:     [][32]byte{{1, 2}},
		OpaqueData: []byte{34, 56},
	}))
	t.Require().NoError(err)
	ref := l2A.EL.BlockRefByNumber(initTx.Receipt.BlockNumber.Uint64())
	out := new(txintent.InteropOutput)
	t.Require().NoError(out.FromReceipt(t.Ctx(), initTx.Receipt, ref.BlockRef(), l2A.EL.ChainID()))
	t.Require().Len(out.Entries, 1)
	validInitMsg := out.Entries[0]

	ctxInvalid, cancelInvalid := context.WithCancel(t.Ctx())
	defer cancelInvalid()
	var wg sync.WaitGroup
	defer wg.Wait()

	// Spam a fixed number of invalid messages per block time.
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := t.WithCtx(ctxInvalid)
		mps := uint64(1_000)
		if mpsStr, exists := os.LookupEnv("NAT_INVALID_MPS"); exists {
			var err error
			mps, err = strconv.ParseUint(mpsStr, 10, 64)
			t.Require().NoError(err)
		}
		NewConstant(l2B.BlockTime, WithBaseRPS(mps)).Run(t, NewInvalidExecMsgSpammer(t, l2B, validInitMsg))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancelInvalid()
		observer := aimdObserver(l2B.EL.ChainID())
		s := NewSteady(l2B.EL.Escape().EthClient(), l2B.Config.ElasticityMultiplier(), l2B.BlockTime, WithAIMDObserver(observer))
		s.Run(t, NewRelaySpammer(l2A, l2B))
	}()
}

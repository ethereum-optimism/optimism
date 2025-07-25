package loadtest

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type messageSource interface {
	Get() *suptypes.Message
}

type messageSink interface {
	Append(*suptypes.Message)
}

type messagePool struct {
	mu       sync.Mutex // mu protects both rng's rand.Source and messages.
	rng      *rand.Rand
	messages []*suptypes.Message
}

var _ messageSource = (*messagePool)(nil)
var _ messageSink = (*messagePool)(nil)

// newMessagePool creates a messagePool, using message as the first message in the pool.
func newMessagePool(message *suptypes.Message) *messagePool {
	return &messagePool{
		rng:      rand.New(rand.NewSource(1234)),
		messages: []*suptypes.Message{message},
	}
}

// Get pseudorandomly selects a message from the pool.
func (p *messagePool) Get() *suptypes.Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	index := p.rng.Intn(len(p.messages))
	return p.messages[index]
}

// Append adds msg to the pool.
func (p *messagePool) Append(msg *suptypes.Message) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messages = append(p.messages, msg)
}

// ExecMsgSpammer is a spammer that executes messages from a source and sends those executing
// messages as initiating messages to a sink.
type ExecMsgSpammer struct {
	l2     *L2
	source messageSource
	sink   messageSink
}

var _ Spammer = (*ExecMsgSpammer)(nil)

func NewExecMsgSpammer(source messageSource, sink messageSink, dest *L2) *ExecMsgSpammer {
	return &ExecMsgSpammer{
		l2:     dest,
		source: source,
		sink:   sink,
	}
}

func (e *ExecMsgSpammer) Spam(t devtest.T) error {
	// Get an initiating message from the source and execute it.
	initMsg := e.source.Get()
	start := time.Now()
	tx, err := e.l2.Include(t, planExecMsg(t, initMsg, e.l2.BlockTime, e.l2.EL.Escape().EthClient()))
	if err != nil {
		return fmt.Errorf("include exec msg: %w", err)
	}
	messageLatency.WithLabelValues(e.l2.Config.ChainID.String(), "exec").Observe(time.Since(start).Seconds())

	// All executing messages are initiating messages. Send the executing message to the sink as
	// an initiating message.
	initMsg, err = initMsgFromReceipt(t, e.l2, tx.Receipt)
	if err != nil {
		return err
	}
	e.sink.Append(initMsg)

	return nil
}

// TestMaxExecutingMessagesBurst runs the ExecMsgSpammer on a Burst schedule on both chains. The
// executing messages emitted by one spammer become initiating messages for the other spammer. The
// test aims to maximize load on the supervisor (indexing and access list checks).
func TestMaxExecutingMessagesBurst(gt *testing.T) {
	t, l2A, l2B := setupLoadTest(gt)

	// Initiate messages on both chains.
	var initMsgFromA, initMsgFromB *suptypes.Message
	func() {
		var wg sync.WaitGroup
		defer wg.Wait()
		wg.Add(1)
		go func() {
			defer wg.Done()
			initMsgFromA = initiate(t, l2A)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			initMsgFromB = initiate(t, l2B)
		}()
	}()
	initMsgsFromA := newMessagePool(initMsgFromA)
	initMsgsFromB := newMessagePool(initMsgFromB)

	// Execute the messages on both chains.
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		NewBurst(l2A.BlockTime, WithAIMDObserver(aimdObserver(l2A.EL.ChainID()))).Run(t, NewExecMsgSpammer(initMsgsFromB, initMsgsFromA, l2A))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		NewBurst(l2B.BlockTime, WithAIMDObserver(aimdObserver(l2B.EL.ChainID()))).Run(t, NewExecMsgSpammer(initMsgsFromA, initMsgsFromB, l2B))
	}()
}

func initiate(t devtest.T, l2 *L2) *suptypes.Message {
	tx, err := l2.Include(t, planCall(t, &txintent.InitTrigger{
		Emitter: l2.EventLogger,
	}))
	t.Require().NoError(err)
	msg, err := initMsgFromReceipt(t, l2, tx.Receipt)
	t.Require().NoError(err)
	return msg
}

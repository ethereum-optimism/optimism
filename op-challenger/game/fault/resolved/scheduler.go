package resolved

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/log"
)

type Validator func(ctx context.Context, block uint64, games []types.GameMetadata) error

type ValidatorScheduler struct {
	log        log.Logger
	ch         chan schedulerMessage
	validators []Validator
	cancel     func()
	wg         sync.WaitGroup
}

type schedulerMessage struct {
	blockNumber uint64
	games       []types.GameMetadata
}

func NewValidatorScheduler(logger log.Logger, validators ...Validator) *ValidatorScheduler {
	return &ValidatorScheduler{
		log:        logger,
		ch:         make(chan schedulerMessage, 1),
		validators: validators,
	}
}

func (v *ValidatorScheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	v.cancel = cancel
	v.wg.Add(1)
	go v.run(ctx)
}

func (v *ValidatorScheduler) Close() error {
	v.cancel()
	v.wg.Wait()
	return nil
}

func (v *ValidatorScheduler) run(ctx context.Context) {
	defer v.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-v.ch:
			v.validate(ctx, msg)
		}
	}
}

func (v *ValidatorScheduler) validate(ctx context.Context, msg schedulerMessage) {
	for _, validator := range v.validators {
		if err := validator(ctx, msg.blockNumber, msg.games); err != nil {
			v.log.Error("Failed to validate game", "blockNumber", msg.blockNumber, "err", err)
		}
	}
}

func (v *ValidatorScheduler) Schedule(blockNumber uint64, games []types.GameMetadata) error {
	select {
	case v.ch <- schedulerMessage{blockNumber, games}:
	default:
		v.log.Trace("Skipping validation while validators in progress")
	}
	return nil
}

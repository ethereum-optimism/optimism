package challenger

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrMissingClient = errors.New("missing client")
)

// SubscriptionId is a unique subscription ID.
type SubscriptionId uint64

// Increment returns the next subscription ID.
func (s *SubscriptionId) Increment() SubscriptionId {
	*s++
	return *s
}

// Subscription wraps an [ethereum.Subscription] to provide a restart.
type Subscription struct {
	// The subscription ID
	id SubscriptionId
	// The current subscription
	sub ethereum.Subscription
	// The query used to create the subscription
	query ethereum.FilterQuery
	// The log channel
	logs <-chan types.Log
	// Filter client used to open the log subscription
	client ethereum.LogFilterer
}

// Subscribe constructs the subscription.
func (s *Subscription) Subscribe() error {
	log.Info("Subscribing to", "query", s.query.Topics, "id", s.id)
	logs := make(chan types.Log)
	if s.client == nil {
		log.Error("missing client")
		return ErrMissingClient
	}
	sub, err := s.client.SubscribeFilterLogs(context.Background(), s.query, logs)
	if err != nil {
		log.Error("failed to subscribe to logs", "err", err)
		return err
	}
	s.sub = sub
	s.logs = logs
	return nil
}

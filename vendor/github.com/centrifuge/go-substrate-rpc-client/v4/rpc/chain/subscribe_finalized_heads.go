// Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
//
// Copyright 2019 Centrifuge GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chain

import (
	"context"
	"sync"

	"github.com/centrifuge/go-substrate-rpc-client/v4/config"
	gethrpc "github.com/centrifuge/go-substrate-rpc-client/v4/gethrpc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

// FinalizedHeadsSubscription is a subscription established through one of the Client's subscribe methods.
type FinalizedHeadsSubscription struct {
	sub      *gethrpc.ClientSubscription
	channel  chan types.Header
	quitOnce sync.Once // ensures quit is closed once
}

// Chan returns the subscription channel.
//
// The channel is closed when Unsubscribe is called on the subscription.
func (s *FinalizedHeadsSubscription) Chan() <-chan types.Header {
	return s.channel
}

// Err returns the subscription error channel. The intended use of Err is to schedule
// resubscription when the client connection is closed unexpectedly.
//
// The error channel receives a value when the subscription has ended due
// to an error. The received error is nil if Close has been called
// on the underlying client and no other error has occurred.
//
// The error channel is closed when Unsubscribe is called on the subscription.
func (s *FinalizedHeadsSubscription) Err() <-chan error {
	return s.sub.Err()
}

// Unsubscribe unsubscribes the notification and closes the error channel.
// It can safely be called more than once.
func (s *FinalizedHeadsSubscription) Unsubscribe() {
	s.sub.Unsubscribe()
	s.quitOnce.Do(func() {
		close(s.channel)
	})
}

// SubscribeFinalizedHeads subscribes the best finalized headers, returning a subscription that will
// receive server notifications containing the Header.
func (c *chain) SubscribeFinalizedHeads() (*FinalizedHeadsSubscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Default().SubscribeTimeout)
	defer cancel()

	ch := make(chan types.Header)

	sub, err := c.client.Subscribe(ctx, "chain", "subscribeFinalizedHeads", "unsubscribeFinalizedHeads",
		"finalizedHead", ch)
	if err != nil {
		return nil, err
	}

	return &FinalizedHeadsSubscription{sub: sub, channel: ch}, nil
}

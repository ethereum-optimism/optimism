package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/coder/websocket"
)

// ErrWSSubscriptionClosed is the error a WSSubscription ends with once it is closed.
var ErrWSSubscriptionClosed = errors.New("websocket subscription closed")

// WSSubscription is a JSON-RPC subscription in the jsonrpsee style over a websocket: the
// subscribe method and its notifications carry whatever names the server chose, rather than the
// eth_subscribe convention go-ethereum's client expects.
type WSSubscription struct {
	cancel        context.CancelFunc
	conn          *WSClient
	notifications chan json.RawMessage

	endErr error
	mu     sync.Mutex
}

// SubscribeWS dials cfg.URL, calls method without parameters and returns once the server accepts
// the subscription. ctx bounds only the dial and the server's reply; the subscription lasts until
// Close, a read fails, or no message arrives within cfg.ReadTimeout.
func SubscribeWS(ctx context.Context, cfg WSConfig, method string) (*WSSubscription, error) {
	conn, err := DialWS(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", cfg.URL, err)
	}
	if cfg.Log != nil {
		cfg.Log.Info("Subscribing over websocket", "url", cfg.URL, "method", method)
	}
	early, err := requestSubscription(ctx, conn, method)
	if err != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "subscription refused")
		if cfg.Log != nil {
			cfg.Log.Warn("Websocket subscription failed", "url", cfg.URL, "method", method, "err", err)
		}
		return nil, fmt.Errorf("failed to subscribe to %s: %w", method, err)
	}
	if cfg.Log != nil {
		cfg.Log.Info("Subscribed over websocket", "url", cfg.URL, "method", method)
	}
	followCtx, cancel := context.WithCancel(context.Background())
	sub := &WSSubscription{cancel: cancel, conn: conn, notifications: make(chan json.RawMessage)}
	go sub.follow(followCtx, early)
	return sub, nil
}

// requestSubscription sends the subscription request and reads until the server replies,
// returning the results of any notifications that arrived before the reply.
func requestSubscription(ctx context.Context, conn *WSClient, method string) ([]json.RawMessage, error) {
	request, err := json.Marshal(struct {
		ID      int      `json:"id"`
		JSONRPC string   `json:"jsonrpc"`
		Method  string   `json:"method"`
		Params  []string `json:"params"`
	}{ID: 1, JSONRPC: "2.0", Method: method, Params: []string{}})
	if err != nil {
		return nil, err
	}
	if err := conn.Write(ctx, websocket.MessageText, request); err != nil {
		return nil, err
	}
	var early []json.RawMessage
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return nil, err
		}
		message, err := decodeWSSubscriptionMessage(data)
		if err != nil {
			return nil, err
		}
		if !message.isNotification {
			return early, nil
		}
		early = append(early, message.result)
	}
}

// Notifications delivers the result of every notification in the order the server sent them. It
// is closed once the subscription ends, after which Err reports why.
func (s *WSSubscription) Notifications() <-chan json.RawMessage {
	return s.notifications
}

// Err reports why the subscription ended, and is nil while it lasts.
func (s *WSSubscription) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.endErr
}

// Close ends the subscription and closes its connection.
func (s *WSSubscription) Close() {
	s.cancel()
}

func (s *WSSubscription) follow(ctx context.Context, early []json.RawMessage) {
	err := s.deliver(ctx, early)
	if ctx.Err() != nil {
		err = ErrWSSubscriptionClosed
	}
	_ = s.conn.Close(websocket.StatusNormalClosure, "subscription ended")
	s.mu.Lock()
	s.endErr = err
	s.mu.Unlock()
	close(s.notifications)
}

// deliver forwards early and then every notification read from the connection until delivery or
// a read fails, and returns the failure.
func (s *WSSubscription) deliver(ctx context.Context, early []json.RawMessage) error {
	for _, result := range early {
		if err := s.send(ctx, result); err != nil {
			return err
		}
	}
	quietLimit := s.conn.config.ReadTimeout
	for {
		readCtx, cancel := context.WithTimeout(ctx, quietLimit)
		_, data, err := s.conn.Read(readCtx)
		quietLimitReached := err != nil && ctx.Err() == nil && errors.Is(readCtx.Err(), context.DeadlineExceeded)
		cancel()
		if quietLimitReached {
			return fmt.Errorf("no message within %s: %w", quietLimit, err)
		}
		if err != nil {
			return err
		}
		message, err := decodeWSSubscriptionMessage(data)
		if err != nil {
			return err
		}
		if !message.isNotification {
			return fmt.Errorf("unexpected reply %q after the subscription was accepted", data)
		}
		if err := s.send(ctx, message.result); err != nil {
			return err
		}
	}
}

func (s *WSSubscription) send(ctx context.Context, result json.RawMessage) error {
	select {
	case s.notifications <- result:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// wsSubscriptionMessage is one message of a subscription: either the server's reply to the
// subscription request or a notification carrying a result.
type wsSubscriptionMessage struct {
	isNotification bool
	result         json.RawMessage
}

func decodeWSSubscriptionMessage(data []byte) (wsSubscriptionMessage, error) {
	var raw struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		ID     json.RawMessage `json:"id"`
		Params *struct {
			Result json.RawMessage `json:"result"`
		} `json:"params"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return wsSubscriptionMessage{}, fmt.Errorf("undecodable message %q: %w", data, err)
	}
	switch {
	case raw.Error != nil:
		return wsSubscriptionMessage{}, fmt.Errorf("server replied with error %d: %s", raw.Error.Code, raw.Error.Message)
	case raw.Params != nil && len(raw.Params.Result) != 0:
		return wsSubscriptionMessage{isNotification: true, result: raw.Params.Result}, nil
	case len(raw.ID) != 0 && len(raw.Result) != 0:
		return wsSubscriptionMessage{}, nil
	default:
		return wsSubscriptionMessage{}, fmt.Errorf("unexpected message %q", data)
	}
}

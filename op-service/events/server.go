package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/event"
	"github.com/pkg/errors"
)

type Server struct {
	PayloadFeed *event.Feed
}

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	EventStreamMediaType   = "text/event-stream"
	KeepAlive              = "keep-alive"
	PayloadAttributesTopic = "payload_attributes"
)

func respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(httpErrorResp{code, message}); err != nil {
		http.Error(w, message, code)
	}
}

func NewServer() *Server {
	return &Server{
		PayloadFeed: new(event.Feed),
	}
}

func (s *Server) Publish(payload *derive.AttributesWithParent) {
	s.PayloadFeed.Send(payload)
}

func (s *Server) StreamEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Streaming unsupported!")
		return
	}

	// Set up SSE response headers
	w.Header().Set("Content-Type", EventStreamMediaType)
	w.Header().Set("Connection", KeepAlive)

	payloadAttrC := make(chan *derive.AttributesWithParent, 1000)
	payloadAttrSub := s.PayloadFeed.Subscribe(payloadAttrC)
	defer payloadAttrSub.Unsubscribe()

	// Handle each event received and context cancellation.
	// We send a keepalive dummy message immediately to prevent clients
	// stalling while waiting for the first response chunk.
	// After that we send a keepalive dummy message every SECONDS_PER_SLOT
	// to prevent anyone (e.g. proxy servers) from closing connections.
	if err := sendKeepalive(w, flusher); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	keepaliveTicker := time.NewTicker(2 * time.Second)

	for {
		select {
		case attrs := <-payloadAttrC:
			if err := s.sendPayloadAttributes(ctx, w, flusher, attrs); err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
		case <-keepaliveTicker.C:
			if err := sendKeepalive(w, flusher); err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) sendPayloadAttributes(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, attrs *derive.AttributesWithParent) error {
	builderAttrs := attrs.ToBuilderPayloadAttributes()
	jsonBytes, err := json.Marshal(builderAttrs)
	if err != nil {
		return err
	}
	return send(w, flusher, PayloadAttributesTopic, jsonBytes)
}

func sendKeepalive(w http.ResponseWriter, flusher http.Flusher) error {
	return write(w, flusher, ":\n\n")
}

func write(w http.ResponseWriter, flusher http.Flusher, format string, a ...any) error {
	_, err := fmt.Fprintf(w, format, a...)
	if err != nil {
		return errors.Wrap(err, "could not write to response writer")
	}
	flusher.Flush()
	return nil
}

func send(w http.ResponseWriter, flusher http.Flusher, name string, data interface{}) error {
	j, err := json.Marshal(data)
	if err != nil {
		return write(w, flusher, "Could not marshal event to JSON: "+err.Error())
	}
	return write(w, flusher, "event: %s\ndata: %s\n\n", name, string(j))
}

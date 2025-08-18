package synctester

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

var ErrInvalidSessionIDFormat = errors.New("invalid UUID")
var ErrInvalidParams = errors.New("invalid param")

// parseSession inspects the incoming request to determine if it targets a session-specific route.
// If the request path matches the pattern `/chain/{chain_id}/synctest/{uuid}`, it attempts to parse
// the UUID and optional query parameters (`latest`, `safe`, `finalized`) used to initialize the session.
//
// If parsing succeeds, a backend.Session is attached to the request context, and the URL path is
// rewritten to `/chain/{chain_id}/synctest` to enable consistent routing downstream.
//
// If the path does not match the session pattern, the request is returned unchanged.
//
// Expected path format for session routes:
//
//	/chain/{chain_id}/synctest/{session_uuid}
//
// Returns an error if the session UUID is invalid or any query parameter is malformed.
func parseSession(r *http.Request, log log.Logger) (*http.Request, error) {
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(segments) == 4 && segments[0] == "chain" && segments[2] == "synctest" {
		sessionID := segments[3]
		if _, err := uuid.Parse(sessionID); err != nil {
			return r, ErrInvalidSessionIDFormat
		}
		query := r.URL.Query()
		parseParam := func(name string) (uint64, error) {
			raw := query.Get(name)
			if raw == "" {
				log.Warn("Parameter not provided. Defaulting to 0", "param", name)
				return 0, nil
			}
			val, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid value for %q: %w", name, ErrInvalidParams)
			}
			return val, nil
		}
		latest, err := parseParam(eth.Unsafe)
		if err != nil {
			return r, err
		}
		safe, err := parseParam(eth.Safe)
		if err != nil {
			return r, err
		}
		finalized, err := parseParam(eth.Finalized)
		if err != nil {
			return r, err
		}
		session := &backend.Session{
			SessionID: sessionID,
			CurrentState: backend.FCUState{
				Latest:    latest,
				Safe:      safe,
				Finalized: finalized,
			},
			InitialState: backend.FCUState{
				Latest:    latest,
				Safe:      safe,
				Finalized: finalized,
			},
		}
		ctx := backend.WithSession(r.Context(), session)
		// remove uuid path for routing
		r.URL.Path = "/" + strings.Join(segments[:3], "/")
		r = r.WithContext(ctx)
	}
	return r, nil
}

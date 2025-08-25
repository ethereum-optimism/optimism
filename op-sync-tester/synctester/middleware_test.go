package synctester

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newRequest(path string, query url.Values) *http.Request {
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: path, RawQuery: query.Encode()},
		Header: make(http.Header),
	}
	return req
}

func TestParseSession_Valid(t *testing.T) {
	id := uuid.New().String()
	query := url.Values{}
	query.Set(eth.Unsafe, "100")
	query.Set(eth.Safe, "90")
	query.Set(eth.Finalized, "80")

	req := newRequest("/chain/1/synctest/"+id, query)
	newReq, err := parseSession(req, log.New())
	require.NoError(t, err)
	require.NotNil(t, newReq)

	session, ok := backend.SessionFromContext(newReq.Context())
	require.True(t, ok)
	require.NotNil(t, session)
	require.Equal(t, id, session.SessionID)
	require.Equal(t, uint64(100), session.InitialState.Latest)
	require.Equal(t, uint64(90), session.InitialState.Safe)
	require.Equal(t, uint64(80), session.InitialState.Finalized)
	require.Equal(t, session.InitialState.Latest, session.Validated)
	require.Equal(t, session.InitialState, session.CurrentState)
	require.Equal(t, "/chain/1/synctest", newReq.URL.Path)
}

func TestParseSession_DefaultsToZero(t *testing.T) {
	id := uuid.New().String()
	req := newRequest("/chain/1/synctest/"+id, nil)

	newReq, err := parseSession(req, log.New())
	require.NoError(t, err)
	require.NotNil(t, newReq)

	session, ok := backend.SessionFromContext(newReq.Context())
	require.True(t, ok)
	require.NotNil(t, session)
	require.Equal(t, id, session.SessionID)
	require.Equal(t, uint64(0), session.InitialState.Latest)
	require.Equal(t, uint64(0), session.InitialState.Safe)
	require.Equal(t, uint64(0), session.InitialState.Finalized)
	require.Equal(t, session.InitialState.Latest, session.Validated)
	require.Equal(t, session.InitialState, session.CurrentState)
}

func TestParseSession_NoSessionInitialized(t *testing.T) {
	req := newRequest("/chain/1/synctest", nil)

	newReq, err := parseSession(req, log.New())
	require.NoError(t, err)
	require.Same(t, req, newReq)

	_, ok := backend.SessionFromContext(newReq.Context())
	require.False(t, ok)
}

func TestParseSession_InvalidSessionIDFormat(t *testing.T) {
	req := newRequest("/chain/1/synctest/not-a-uuid", nil)
	_, err := parseSession(req, log.New())
	require.ErrorIs(t, err, ErrInvalidSessionIDFormat)
}

func TestParseSession_InvalidQueryParam(t *testing.T) {
	id := uuid.New().String()
	query := url.Values{}
	query.Set(eth.Unsafe, "not-a-number") // invalid uint64

	req := newRequest("/chain/1/synctest/"+id, query)
	_, err := parseSession(req, log.New())
	require.ErrorIs(t, err, ErrInvalidParams)
}

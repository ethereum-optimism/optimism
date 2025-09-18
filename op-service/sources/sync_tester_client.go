package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SyncTesterClient struct {
	client client.RPC
}

var _ apis.SyncTester = (*SyncTesterClient)(nil)

func NewSyncTesterClient(client client.RPC) *SyncTesterClient {
	return &SyncTesterClient{
		client: client,
	}
}

func (cl *SyncTesterClient) ChainID(ctx context.Context) (eth.ChainID, error) {
	var result eth.ChainID
	err := cl.client.CallContext(ctx, &result, "eth_chainId")
	return result, err
}

func (cl *SyncTesterClient) GetSession(ctx context.Context) (*eth.SyncTesterSession, error) {
	var session *eth.SyncTesterSession
	err := cl.client.CallContext(ctx, &session, "sync_getSession")
	return session, err
}

func (cl *SyncTesterClient) ListSessions(ctx context.Context) ([]string, error) {
	var sessions []string
	err := cl.client.CallContext(ctx, &sessions, "sync_listSessions")
	return sessions, err
}

func (cl *SyncTesterClient) DeleteSession(ctx context.Context) error {
	return cl.client.CallContext(ctx, nil, "sync_deleteSession")
}

func (cl *SyncTesterClient) ResetSession(ctx context.Context) error {
	return cl.client.CallContext(ctx, nil, "sync_resetSession")
}

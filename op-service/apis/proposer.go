package apis

import "context"

type ProposerActivity interface {
	StartProposer(ctx context.Context) error
	StopProposer(ctx context.Context) error
}

type ProposerAdminServer interface {
	CommonAdminServer
	ProposerActivity
}

type ProposerAdminClient interface {
	CommonAdminClient
	ProposerActivity
}

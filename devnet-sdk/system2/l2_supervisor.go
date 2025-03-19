package system2

import (
	"github.com/ethereum-optimism/optimism/op-service/client"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// L2SupervisorID identifies a L2Supervisor by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2SupervisorID genericID

func (id L2SupervisorID) String() string {
	return genericID(id).string("L2Supervisor")
}

func (id L2SupervisorID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText("L2Supervisor")
}

func (id *L2SupervisorID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText("L2Supervisor", data)
}

func SortL2SupervisorIDs(ids []L2SupervisorID) []L2SupervisorID {
	return copyAndSortCmp(ids)
}

// L2Supervisor is an interop service, used to cross-verify messages between chains.
type L2Supervisor interface {
	Common
	ID() L2SupervisorID

	AdminAPI() sources.SupervisorAdminAPI
	QueryAPI() sources.SupervisorQueryAPI
}

type L2SupervisorConfig struct {
	CommonConfig
	ID     L2SupervisorID
	Client client.RPC
}

type rpcL2Supervisor struct {
	commonImpl
	id L2SupervisorID

	cl  client.RPC
	api interface {
		sources.SupervisorQueryAPI
		sources.SupervisorAdminAPI
	}
}

var _ L2Supervisor = (*rpcL2Supervisor)(nil)

func NewL2Supervisor(cfg L2SupervisorConfig) L2Supervisor {
	cfg.Log = cfg.Log.New("id", cfg.ID)
	return &rpcL2Supervisor{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		cl:         cfg.Client,
		api:        sources.NewSupervisorClient(cfg.Client, &opmetrics.NoopRPCMetrics{}),
	}
}

func (r *rpcL2Supervisor) ID() L2SupervisorID {
	return r.id
}

func (r *rpcL2Supervisor) AdminAPI() sources.SupervisorAdminAPI {
	return r.api
}

func (r *rpcL2Supervisor) QueryAPI() sources.SupervisorQueryAPI {
	return r.api
}

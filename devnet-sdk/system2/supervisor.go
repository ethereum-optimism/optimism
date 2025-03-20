package system2

import (
	"github.com/ethereum-optimism/optimism/op-service/client"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// SupervisorID identifies a Supervisor by name and chainID, is type-safe, and can be value-copied and used as map key.
type SupervisorID genericID

const SupervisorKind Kind = "Supervisor"

func (id SupervisorID) String() string {
	return genericID(id).string(SupervisorKind)
}

func (id SupervisorID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText(SupervisorKind)
}

func (id *SupervisorID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText(SupervisorKind, data)
}

func SortSupervisorIDs(ids []SupervisorID) []SupervisorID {
	return copyAndSortCmp(ids)
}

// Supervisor is an interop service, used to cross-verify messages between chains.
type Supervisor interface {
	Common
	ID() SupervisorID

	AdminAPI() sources.SupervisorAdminAPI
	QueryAPI() sources.SupervisorQueryAPI
}

type SupervisorConfig struct {
	CommonConfig
	ID     SupervisorID
	Client client.RPC
}

type rpcSupervisor struct {
	commonImpl
	id SupervisorID

	client client.RPC
	api    interface {
		sources.SupervisorQueryAPI
		sources.SupervisorAdminAPI
	}
}

var _ Supervisor = (*rpcSupervisor)(nil)

func NewSupervisor(cfg SupervisorConfig) Supervisor {
	cfg.Log = cfg.Log.New("id", cfg.ID)
	return &rpcSupervisor{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
		api:        sources.NewSupervisorClient(cfg.Client, &opmetrics.NoopRPCMetrics{}),
	}
}

func (r *rpcSupervisor) ID() SupervisorID {
	return r.id
}

func (r *rpcSupervisor) AdminAPI() sources.SupervisorAdminAPI {
	return r.api
}

func (r *rpcSupervisor) QueryAPI() sources.SupervisorQueryAPI {
	return r.api
}

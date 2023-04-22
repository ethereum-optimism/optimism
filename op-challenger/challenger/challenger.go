package challenger

import (
	"context"
	"crypto/ecdsa"
	_ "net/http/pprof"
	"sync"
	"time"

	abi "github.com/ethereum/go-ethereum/accounts/abi"
	common "github.com/ethereum/go-ethereum/common"
	ethclient "github.com/ethereum/go-ethereum/ethclient"
	log "github.com/ethereum/go-ethereum/log"

	bindings "github.com/refcell/op-challenger/contracts/bindings"
	metrics "github.com/refcell/op-challenger/metrics"

	opBindings "github.com/ethereum-optimism/optimism/op-bindings/bindings"
	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	sources "github.com/ethereum-optimism/optimism/op-node/sources"
	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var supportedL2OutputVersion = eth.Bytes32{}

const (
	// FaultDisputeGameType is the uint8 enum value for the fault dispute game
	FaultDisputeGameType = 0
	// ValidityDisputeGameType is the uint8 enum value for the validity dispute game
	ValidityDisputeGameType = 1
	// AttestationDisputeGameType is the uint8 enum value for the attestation dispute game
	AttestationDisputeGameType = 2
)

// Challenger is responsible for disputing L2OutputOracle outputs
type Challenger struct {
	txMgr txmgr.TxManager
	wg    sync.WaitGroup
	done  chan struct{}
	log   log.Logger
	metr  metrics.Metricer

	privateKey *ecdsa.PrivateKey
	from       common.Address

	ctx    context.Context
	cancel context.CancelFunc

	l1Client *ethclient.Client

	rollupClient *sources.RollupClient

	l2ooContract     *opBindings.L2OutputOracleCaller
	l2ooContractAddr common.Address
	l2ooABI          *abi.ABI

	dgfContract     *bindings.MockDisputeGameFactoryCaller
	dgfContractAddr common.Address
	dgfABI          *abi.ABI

	adgABI *abi.ABI

	networkTimeout time.Duration
}

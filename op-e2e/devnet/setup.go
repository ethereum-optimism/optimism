package devnet

import (
	"context"
	"crypto/ecdsa"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/sources"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// TODO(#10968): read from docker-compose.yml
const (
	L1RPCURL  = "http://127.0.0.1:8545"
	L2RPCURL  = "http://127.0.0.1:9545"
	RollupURL = "http://127.0.0.1:7545"
)

type System struct {
	L1     *ethclient.Client
	L2     *ethclient.Client
	Rollup *sources.RollupClient
	Cfg    e2esys.SystemConfig
}

func NewSystem(ctx context.Context, lgr log.Logger) (sys *System, err error) {
	sys = new(System)
	sys.L1, err = dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, lgr, L1RPCURL)
	if err != nil {
		return nil, err
	}
	sys.L2, err = dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, lgr, L2RPCURL)
	if err != nil {
		return nil, err
	}
	sys.Rollup, err = dial.DialRollupClientWithTimeout(ctx, dial.DefaultDialTimeout, lgr, RollupURL)
	if err != nil {
		return nil, err
	}

	secrets, err := e2eutils.DefaultMnemonicConfig.Secrets()
	if err != nil {
		return nil, err
	}

	// TODO(#10968): We need to re-read the deploy config because op-e2e/config.init() overwrites
	// some deploy config variables. This will be fixed soon.
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, err := op_service.FindMonorepoRoot(cwd)
	if err != nil {
		return nil, err
	}
	deployConfigPath := filepath.Join(root, "packages", "contracts-bedrock", "deploy-config", "devnetL1.json")
	deployConfig, err := genesis.NewDeployConfig(deployConfigPath)
	if err != nil {
		return nil, err
	}

	// Incomplete SystemConfig suffices for withdrawal test (only consumer right now)
	sys.Cfg = e2esys.SystemConfig{
		DeployConfig:  deployConfig,
		L1Deployments: config.L1Deployments.Copy(),
		Secrets:       secrets,
	}
	return sys, nil
}

func (s System) NodeClient(role string) *ethclient.Client {
	switch role {
	case e2esys.RoleL1:
		return s.L1
	case e2esys.RoleSeq, e2esys.RoleVerif:
		// we have only one L2 node
		return s.L2
	default:
		panic("devnet.System: unknown role: " + role)
	}
}

func (s System) RollupClient(string) *sources.RollupClient {
	// we ignore role, have only one L2 rollup
	return s.Rollup
}

func (s System) Config() e2esys.SystemConfig {
	return s.Cfg
}

func (s System) TestAccount(idx int) *ecdsa.PrivateKey {
	// first 12 indices are in use by the devnet
	return s.Cfg.Secrets.AccountAtIdx(13 + idx)
}

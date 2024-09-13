package deployer

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/kurtosisutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const TestParams = `
participants:
  - el_type: geth
    cl_type: lighthouse
network_params:
  prefunded_accounts: '{ "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266": { "balance": "1000000ETH" } }'
  network_id: "77799777"
  seconds_per_slot: 3
`

type deployerKey struct{}

func (d *deployerKey) HDPath() string {
	return "m/44'/60'/0'/0/0"
}

func (d *deployerKey) String() string {
	return "deployer-key"
}

func TestEndToEndApply(t *testing.T) {
	kurtosisutil.Test(t)

	lgr := testlog.Logger(t, slog.LevelInfo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wd, err := os.Getwd()
	require.NoError(t, err)
	monorepoDir := path.Join(wd, "..", "..")
	artifactsDir := path.Join(monorepoDir, "packages", "contracts-bedrock", "forge-artifacts")
	require.NoError(t, err)

	enclaveCtx := kurtosisutil.StartEnclave(t, ctx, lgr, "github.com/ethpandaops/ethereum-package", TestParams)

	service, err := enclaveCtx.GetServiceContext("el-1-geth-lighthouse")
	require.NoError(t, err)

	ip := service.GetMaybePublicIPAddress()
	ports := service.GetPublicPorts()
	rpcURL := fmt.Sprintf("http://%s:%d", ip, ports["rpc"].GetNumber())
	l1Client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)

	artifactsURL, err := url.Parse(fmt.Sprintf("file://%s", artifactsDir))
	require.NoError(t, err)

	depKey := new(deployerKey)
	l1ChainID := big.NewInt(77799777)
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	pk, err := dk.Secret(depKey)
	require.NoError(t, err)
	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(pk, l1ChainID))

	addrFor := func(key devkeys.Key) common.Address {
		addr, err := dk.Address(key)
		require.NoError(t, err)
		return addr
	}
	env := &pipeline.Env{
		Workdir:  t.TempDir(),
		L1Client: l1Client,
		Signer:   signer,
		Deployer: addrFor(depKey),
		Logger:   lgr,
	}
	intent := &state.Intent{
		L1ChainID: l1ChainID.Uint64(),
		SuperchainRoles: state.SuperchainRoles{
			ProxyAdminOwner:       addrFor(devkeys.L1ProxyAdminOwnerRole.Key(l1ChainID)),
			ProtocolVersionsOwner: addrFor(devkeys.SuperchainDeployerKey.Key(l1ChainID)),
			Guardian:              addrFor(devkeys.SuperchainConfigGuardianKey.Key(l1ChainID)),
		},
		UseFaultProofs:       true,
		FundDevAccounts:      true,
		ContractArtifactsURL: (*state.ArtifactsURL)(artifactsURL),
	}
	st := &state.State{
		Version: 1,
	}

	require.NoError(t, ApplyPipeline(
		ctx,
		env,
		intent,
		st,
	))

	addrs := []common.Address{
		st.SuperchainDeployment.ProxyAdminAddress,
		st.SuperchainDeployment.SuperchainConfigProxyAddress,
		st.SuperchainDeployment.SuperchainConfigImplAddress,
		st.SuperchainDeployment.ProtocolVersionsProxyAddress,
		st.SuperchainDeployment.ProtocolVersionsImplAddress,
	}
	for _, addr := range addrs {
		code, err := l1Client.CodeAt(ctx, addr, nil)
		require.NoError(t, err)
		require.NotEmpty(t, code)
	}
}

package pipeline

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	state2 "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func DeployOPChain(ctx context.Context, env *Env, bundle ArtifactsBundle, intent *state2.Intent, st *state2.State, chainID common.Hash) error {
	lgr := env.Logger.New("stage", "deploy-opchain")

	if !shouldDeployOPChain(intent, st, chainID) {
		lgr.Info("opchain deployment not needed")
		return nil
	}

	lgr.Info("deploying OP chain", "id", chainID.Hex())

	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain intent: %w", err)
	}

	input := opcm.DeployOPChainInput{
		OpChainProxyAdminOwner:  thisIntent.Roles.ProxyAdminOwner,
		SystemConfigOwner:       thisIntent.Roles.SystemConfigOwner,
		Batcher:                 thisIntent.Roles.Batcher,
		UnsafeBlockSigner:       thisIntent.Roles.UnsafeBlockSigner,
		Proposer:                thisIntent.Roles.Proposer,
		Challenger:              thisIntent.Roles.Challenger,
		BasefeeScalar:           1368,
		BlobBaseFeeScalar:       801949,
		L2ChainId:               chainID.Big(),
		OpcmProxy:               st.ImplementationsDeployment.OpcmProxyAddress,
		SaltMixer:               st.Create2Salt.String(), // passing through salt generated at state initialization
		GasLimit:                30_000_000,
		DisputeGameType:         1, // PERMISSIONED_CANNON Game Type
		DisputeAbsolutePrestate: common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
		DisputeMaxGameDepth:     73,
		DisputeSplitDepth:       30,
		DisputeClockExtension:   10800,  // 3 hours (input in seconds)
		DisputeMaxClockDuration: 302400, // 3.5 days (input in seconds)
	}

	var dco opcm.DeployOPChainOutput
	lgr.Info("deploying using existing OPCM", "address", st.ImplementationsDeployment.OpcmProxyAddress.Hex())
	bcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  lgr,
		ChainID: big.NewInt(int64(intent.L1ChainID)),
		Client:  env.L1Client,
		Signer:  env.Signer,
		From:    env.Deployer,
	})
	if err != nil {
		return fmt.Errorf("failed to create broadcaster: %w", err)
	}
	dco, err = opcm.DeployOPChainRaw(
		ctx,
		env.L1Client,
		bcaster,
		env.Deployer,
		bundle.L1,
		input,
	)
	if err != nil {
		return fmt.Errorf("error deploying OP chain: %w", err)
	}

	st.Chains = append(st.Chains, &state2.ChainState{
		ID:                                        chainID,
		ProxyAdminAddress:                         dco.OpChainProxyAdmin,
		AddressManagerAddress:                     dco.AddressManager,
		L1ERC721BridgeProxyAddress:                dco.L1ERC721BridgeProxy,
		SystemConfigProxyAddress:                  dco.SystemConfigProxy,
		OptimismMintableERC20FactoryProxyAddress:  dco.OptimismMintableERC20FactoryProxy,
		L1StandardBridgeProxyAddress:              dco.L1StandardBridgeProxy,
		L1CrossDomainMessengerProxyAddress:        dco.L1CrossDomainMessengerProxy,
		OptimismPortalProxyAddress:                dco.OptimismPortalProxy,
		DisputeGameFactoryProxyAddress:            dco.DisputeGameFactoryProxy,
		AnchorStateRegistryProxyAddress:           dco.AnchorStateRegistryProxy,
		AnchorStateRegistryImplAddress:            dco.AnchorStateRegistryImpl,
		FaultDisputeGameAddress:                   dco.FaultDisputeGame,
		PermissionedDisputeGameAddress:            dco.PermissionedDisputeGame,
		DelayedWETHPermissionedGameProxyAddress:   dco.DelayedWETHPermissionedGameProxy,
		DelayedWETHPermissionlessGameProxyAddress: dco.DelayedWETHPermissionlessGameProxy,
	})

	block, err := env.L1Client.BlockByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest block by number: %w", err)
	}
	currentBlockHash := block.Hash()

	errCh := make(chan error, 8)

	// If any of the implementations addresses (excluding OpcmProxy) are empty,
	// we need to set them using the implementation address read from their corresponding proxy.
	// The reason these might be empty is because we're only invoking DeployOPChain.s.sol as part of the pipeline.
	// TODO: Need to initialize 'mipsSingletonAddress' and 'preimageOracleSingletonAddress'
	setImplementationAddressTasks := []func(){
		func() {
			setEIP1967ImplementationAddress(ctx, env.L1Client, errCh, dco.DelayedWETHPermissionedGameProxy, currentBlockHash, &st.ImplementationsDeployment.DelayedWETHImplAddress)
		},
		func() {
			setEIP1967ImplementationAddress(ctx, env.L1Client, errCh, dco.OptimismPortalProxy, currentBlockHash, &st.ImplementationsDeployment.OptimismPortalImplAddress)
		},
		func() {
			setEIP1967ImplementationAddress(ctx, env.L1Client, errCh, dco.SystemConfigProxy, currentBlockHash, &st.ImplementationsDeployment.SystemConfigImplAddress)
		},
		func() {
			setRDPImplementationAddress(ctx, env.L1Client, errCh, dco.AddressManager, &st.ImplementationsDeployment.L1CrossDomainMessengerImplAddress)
		},
		func() {
			setEIP1967ImplementationAddress(ctx, env.L1Client, errCh, dco.L1ERC721BridgeProxy, currentBlockHash, &st.ImplementationsDeployment.L1ERC721BridgeImplAddress)
		},
		func() {
			setEIP1967ImplementationAddress(ctx, env.L1Client, errCh, dco.L1StandardBridgeProxy, currentBlockHash, &st.ImplementationsDeployment.L1StandardBridgeImplAddress)
		},
		func() {
			setEIP1967ImplementationAddress(ctx, env.L1Client, errCh, dco.OptimismMintableERC20FactoryProxy, currentBlockHash, &st.ImplementationsDeployment.OptimismMintableERC20FactoryImplAddress)
		},
		func() {
			setEIP1967ImplementationAddress(ctx, env.L1Client, errCh, dco.DisputeGameFactoryProxy, currentBlockHash, &st.ImplementationsDeployment.DisputeGameFactoryImplAddress)
		},
	}
	for _, task := range setImplementationAddressTasks {
		go task()
	}

	var lastTaskErr error
	for i := 0; i < len(setImplementationAddressTasks); i++ {
		taskErr := <-errCh
		if taskErr != nil {
			lastTaskErr = taskErr
		}
	}
	if lastTaskErr != nil {
		return fmt.Errorf("failed to set implementation addresses: %w", lastTaskErr)
	}

	return nil
}

func setRDPImplementationAddress(ctx context.Context, client *ethclient.Client, errCh chan error, addressManager common.Address, implAddress *common.Address) {
	if *implAddress != (common.Address{}) {
		errCh <- nil
		return
	}

	contract := opcm.NewContract(addressManager, client)
	address, err := contract.GetAddressByName(ctx, "OVM_L1CrossDomainMessenger")
	if err == nil {
		*implAddress = address
	}
	errCh <- err
}

func setEIP1967ImplementationAddress(ctx context.Context, client *ethclient.Client, errCh chan error, proxy common.Address, currentBlockHash common.Hash, implAddress *common.Address) {
	if *implAddress != (common.Address{}) {
		errCh <- nil
		return
	}

	storageValue, err := client.StorageAtHash(ctx, proxy, genesis.ImplementationSlot, currentBlockHash)
	if err == nil {
		*implAddress = common.HexToAddress(hex.EncodeToString(storageValue))
	}
	errCh <- err
}

func shouldDeployOPChain(intent *state.Intent, st *state.State, chainID common.Hash) bool {
	for _, chain := range st.Chains {
		if chain.ID == chainID {
			return false
		}
	}

	return true
}

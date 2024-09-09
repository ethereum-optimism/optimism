// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console2 as console } from "forge-std/console2.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { L2CrossDomainMessenger } from "src/L2/L2CrossDomainMessenger.sol";
import { L2StandardBridgeInterop } from "src/L2/L2StandardBridgeInterop.sol";
import { L2ToL1MessagePasser } from "src/L2/L2ToL1MessagePasser.sol";
import { L2ERC721Bridge } from "src/L2/L2ERC721Bridge.sol";
import { BaseFeeVault } from "src/L2/BaseFeeVault.sol";
import { SequencerFeeVault } from "src/L2/SequencerFeeVault.sol";
import { L1FeeVault } from "src/L2/L1FeeVault.sol";
import { GasPriceOracle } from "src/L2/GasPriceOracle.sol";
import { L1Block } from "src/L2/L1Block.sol";
import { LegacyMessagePasser } from "src/legacy/LegacyMessagePasser.sol";
import { GovernanceToken } from "src/governance/GovernanceToken.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { StandardBridge } from "src/universal/StandardBridge.sol";
import { FeeVault } from "src/universal/FeeVault.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { DeployConfig } from "scripts/deploy/DeployConfig.s.sol";
import { Deploy } from "scripts/deploy/Deploy.s.sol";
import { Fork, LATEST_FORK } from "scripts/libraries/Config.sol";
import { L2Genesis, L1Dependencies } from "scripts/L2Genesis.s.sol";
import { OutputMode, Fork, ForkUtils } from "scripts/libraries/Config.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { AddressManager } from "src/legacy/AddressManager.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { Executables } from "scripts/libraries/Executables.sol";
import { Vm } from "forge-std/Vm.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { DataAvailabilityChallenge } from "src/L1/DataAvailabilityChallenge.sol";
import { WETH } from "src/L2/WETH.sol";
import { SuperchainWETH } from "src/L2/SuperchainWETH.sol";
import { ETHLiquidity } from "src/L2/ETHLiquidity.sol";

/// @title Setup
/// @dev This contact is responsible for setting up the contracts in state. It currently
///      sets the L2 contracts directly at the predeploy addresses instead of setting them
///      up behind proxies. In the future we will migrate to importing the genesis JSON
///      file that is created to set up the L2 contracts instead of setting them up manually.
contract Setup {
    using ForkUtils for Fork;

    /// @notice The address of the foundry Vm contract.
    Vm private constant vm = Vm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    /// @notice The address of the Deploy contract. Set into state with `etch` to avoid
    ///         mutating any nonces. MUST not have constructor logic.
    Deploy internal constant deploy = Deploy(address(uint160(uint256(keccak256(abi.encode("optimism.deploy"))))));

    L2Genesis internal constant l2Genesis =
        L2Genesis(address(uint160(uint256(keccak256(abi.encode("optimism.l2genesis"))))));

    // @notice Allows users of Setup to override what L2 genesis is being created.
    Fork l2Fork = LATEST_FORK;

    OptimismPortal optimismPortal;
    OptimismPortal2 optimismPortal2;
    DisputeGameFactory disputeGameFactory;
    DelayedWETH delayedWeth;
    L2OutputOracle l2OutputOracle;
    SystemConfig systemConfig;
    L1StandardBridge l1StandardBridge;
    L1CrossDomainMessenger l1CrossDomainMessenger;
    AddressManager addressManager;
    L1ERC721Bridge l1ERC721Bridge;
    OptimismMintableERC20Factory l1OptimismMintableERC20Factory;
    ProtocolVersions protocolVersions;
    SuperchainConfig superchainConfig;
    DataAvailabilityChallenge dataAvailabilityChallenge;
    AnchorStateRegistry anchorStateRegistry;

    L2CrossDomainMessenger l2CrossDomainMessenger =
        L2CrossDomainMessenger(payable(Predeploys.L2_CROSS_DOMAIN_MESSENGER));
    L2StandardBridgeInterop l2StandardBridge = L2StandardBridgeInterop(payable(Predeploys.L2_STANDARD_BRIDGE));
    L2ToL1MessagePasser l2ToL1MessagePasser = L2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER));
    OptimismMintableERC20Factory l2OptimismMintableERC20Factory =
        OptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY);
    L2ERC721Bridge l2ERC721Bridge = L2ERC721Bridge(Predeploys.L2_ERC721_BRIDGE);
    BaseFeeVault baseFeeVault = BaseFeeVault(payable(Predeploys.BASE_FEE_VAULT));
    SequencerFeeVault sequencerFeeVault = SequencerFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET));
    L1FeeVault l1FeeVault = L1FeeVault(payable(Predeploys.L1_FEE_VAULT));
    GasPriceOracle gasPriceOracle = GasPriceOracle(Predeploys.GAS_PRICE_ORACLE);
    L1Block l1Block = L1Block(Predeploys.L1_BLOCK_ATTRIBUTES);
    LegacyMessagePasser legacyMessagePasser = LegacyMessagePasser(Predeploys.LEGACY_MESSAGE_PASSER);
    GovernanceToken governanceToken = GovernanceToken(Predeploys.GOVERNANCE_TOKEN);
    WETH weth = WETH(payable(Predeploys.WETH));
    SuperchainWETH superchainWeth = SuperchainWETH(payable(Predeploys.SUPERCHAIN_WETH));
    ETHLiquidity ethLiquidity = ETHLiquidity(Predeploys.ETH_LIQUIDITY);

    /// @dev Deploys the Deploy contract without including its bytecode in the bytecode
    ///      of this contract by fetching the bytecode dynamically using `vm.getCode()`.
    ///      If the Deploy bytecode is included in this contract, then it will double
    ///      the compile time and bloat all of the test contract artifacts since they
    ///      will also need to include the bytecode for the Deploy contract.
    ///      This is a hack as we are pushing solidity to the edge.
    function setUp() public virtual {
        console.log("L1 setup start!");
        vm.etch(address(deploy), vm.getDeployedCode("Deploy.s.sol:Deploy"));
        vm.allowCheatcodes(address(deploy));
        deploy.setUp();
        console.log("L1 setup done!");

        console.log("L2 setup start!");
        vm.etch(address(l2Genesis), vm.getDeployedCode("L2Genesis.s.sol:L2Genesis"));
        vm.allowCheatcodes(address(l2Genesis));
        l2Genesis.setUp();
        console.log("L2 setup done!");
    }

    /// @dev Sets up the L1 contracts.
    function L1() public {
        console.log("Setup: creating L1 deployments");
        // Set the deterministic deployer in state to ensure that it is there
        vm.etch(
            0x4e59b44847b379578588920cA78FbF26c0B4956C,
            hex"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf3"
        );

        deploy.run();
        console.log("Setup: completed L1 deployment, registering addresses now");

        optimismPortal = OptimismPortal(deploy.mustGetAddress("OptimismPortalProxy"));
        optimismPortal2 = OptimismPortal2(deploy.mustGetAddress("OptimismPortalProxy"));
        disputeGameFactory = DisputeGameFactory(deploy.mustGetAddress("DisputeGameFactoryProxy"));
        delayedWeth = DelayedWETH(deploy.mustGetAddress("DelayedWETHProxy"));
        l2OutputOracle = L2OutputOracle(deploy.mustGetAddress("L2OutputOracleProxy"));
        systemConfig = SystemConfig(deploy.mustGetAddress("SystemConfigProxy"));
        l1StandardBridge = L1StandardBridge(deploy.mustGetAddress("L1StandardBridgeProxy"));
        l1CrossDomainMessenger = L1CrossDomainMessenger(deploy.mustGetAddress("L1CrossDomainMessengerProxy"));
        addressManager = AddressManager(deploy.mustGetAddress("AddressManager"));
        l1ERC721Bridge = L1ERC721Bridge(deploy.mustGetAddress("L1ERC721BridgeProxy"));
        l1OptimismMintableERC20Factory =
            OptimismMintableERC20Factory(deploy.mustGetAddress("OptimismMintableERC20FactoryProxy"));
        protocolVersions = ProtocolVersions(deploy.mustGetAddress("ProtocolVersionsProxy"));
        superchainConfig = SuperchainConfig(deploy.mustGetAddress("SuperchainConfigProxy"));
        anchorStateRegistry = AnchorStateRegistry(deploy.mustGetAddress("AnchorStateRegistryProxy"));

        vm.label(address(l2OutputOracle), "L2OutputOracle");
        vm.label(deploy.mustGetAddress("L2OutputOracleProxy"), "L2OutputOracleProxy");
        vm.label(address(optimismPortal), "OptimismPortal");
        vm.label(deploy.mustGetAddress("OptimismPortalProxy"), "OptimismPortalProxy");
        vm.label(address(disputeGameFactory), "DisputeGameFactory");
        vm.label(deploy.mustGetAddress("DisputeGameFactoryProxy"), "DisputeGameFactoryProxy");
        vm.label(address(delayedWeth), "DelayedWETH");
        vm.label(deploy.mustGetAddress("DelayedWETHProxy"), "DelayedWETHProxy");
        vm.label(address(systemConfig), "SystemConfig");
        vm.label(deploy.mustGetAddress("SystemConfigProxy"), "SystemConfigProxy");
        vm.label(address(l1StandardBridge), "L1StandardBridge");
        vm.label(deploy.mustGetAddress("L1StandardBridgeProxy"), "L1StandardBridgeProxy");
        vm.label(address(l1CrossDomainMessenger), "L1CrossDomainMessenger");
        vm.label(deploy.mustGetAddress("L1CrossDomainMessengerProxy"), "L1CrossDomainMessengerProxy");
        vm.label(address(addressManager), "AddressManager");
        vm.label(address(l1ERC721Bridge), "L1ERC721Bridge");
        vm.label(deploy.mustGetAddress("L1ERC721BridgeProxy"), "L1ERC721BridgeProxy");
        vm.label(address(l1OptimismMintableERC20Factory), "OptimismMintableERC20Factory");
        vm.label(deploy.mustGetAddress("OptimismMintableERC20FactoryProxy"), "OptimismMintableERC20FactoryProxy");
        vm.label(address(protocolVersions), "ProtocolVersions");
        vm.label(deploy.mustGetAddress("ProtocolVersionsProxy"), "ProtocolVersionsProxy");
        vm.label(address(superchainConfig), "SuperchainConfig");
        vm.label(deploy.mustGetAddress("SuperchainConfigProxy"), "SuperchainConfigProxy");
        vm.label(AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)), "L1CrossDomainMessenger_aliased");

        if (deploy.cfg().useAltDA()) {
            dataAvailabilityChallenge =
                DataAvailabilityChallenge(deploy.mustGetAddress("DataAvailabilityChallengeProxy"));
            vm.label(address(dataAvailabilityChallenge), "DataAvailabilityChallengeProxy");
            vm.label(deploy.mustGetAddress("DataAvailabilityChallenge"), "DataAvailabilityChallenge");
        }
        console.log("Setup: registered L1 deployments");
    }

    /// @dev Sets up the L2 contracts. Depends on `L1()` being called first.
    function L2() public {
        console.log("Setup: creating L2 genesis with fork %s", l2Fork.toString());
        l2Genesis.runWithOptions(
            OutputMode.NONE,
            l2Fork,
            L1Dependencies({
                l1CrossDomainMessengerProxy: payable(address(l1CrossDomainMessenger)),
                l1StandardBridgeProxy: payable(address(l1StandardBridge)),
                l1ERC721BridgeProxy: payable(address(l1ERC721Bridge))
            })
        );

        // Set the governance token's owner to be the final system owner
        address finalSystemOwner = deploy.cfg().finalSystemOwner();
        vm.startPrank(governanceToken.owner());
        governanceToken.transferOwnership(finalSystemOwner);
        vm.stopPrank();

        // L2 predeploys
        labelPredeploy(Predeploys.L2_STANDARD_BRIDGE);
        labelPredeploy(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        labelPredeploy(Predeploys.L2_TO_L1_MESSAGE_PASSER);
        labelPredeploy(Predeploys.SEQUENCER_FEE_WALLET);
        labelPredeploy(Predeploys.L2_ERC721_BRIDGE);
        labelPredeploy(Predeploys.BASE_FEE_VAULT);
        labelPredeploy(Predeploys.L1_FEE_VAULT);
        labelPredeploy(Predeploys.L1_BLOCK_ATTRIBUTES);
        labelPredeploy(Predeploys.GAS_PRICE_ORACLE);
        labelPredeploy(Predeploys.LEGACY_MESSAGE_PASSER);
        labelPredeploy(Predeploys.GOVERNANCE_TOKEN);
        labelPredeploy(Predeploys.EAS);
        labelPredeploy(Predeploys.SCHEMA_REGISTRY);
        labelPredeploy(Predeploys.WETH);
        labelPredeploy(Predeploys.SUPERCHAIN_WETH);
        labelPredeploy(Predeploys.ETH_LIQUIDITY);

        // L2 Preinstalls
        labelPreinstall(Preinstalls.MultiCall3);
        labelPreinstall(Preinstalls.Create2Deployer);
        labelPreinstall(Preinstalls.Safe_v130);
        labelPreinstall(Preinstalls.SafeL2_v130);
        labelPreinstall(Preinstalls.MultiSendCallOnly_v130);
        labelPreinstall(Preinstalls.SafeSingletonFactory);
        labelPreinstall(Preinstalls.DeterministicDeploymentProxy);
        labelPreinstall(Preinstalls.MultiSend_v130);
        labelPreinstall(Preinstalls.Permit2);
        labelPreinstall(Preinstalls.SenderCreator_v060);
        labelPreinstall(Preinstalls.EntryPoint_v060);
        labelPreinstall(Preinstalls.SenderCreator_v070);
        labelPreinstall(Preinstalls.EntryPoint_v070);
        labelPreinstall(Preinstalls.BeaconBlockRoots);
        labelPreinstall(Preinstalls.CreateX);

        console.log("Setup: completed L2 genesis");
    }

    function labelPredeploy(address _addr) internal {
        vm.label(_addr, Predeploys.getName(_addr));
    }

    function labelPreinstall(address _addr) internal {
        vm.label(_addr, Preinstalls.getName(_addr));
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Proxy } from "src/universal/Proxy.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { StdAssertions } from "forge-std/StdAssertions.sol";
import "src/libraries/DisputeTypes.sol";
import "scripts/Deploy.s.sol";

/// @notice Deploys the Fault Proof Alpha Chad contracts.
contract FPACOPS is Deploy, StdAssertions {
    ////////////////////////////////////////////////////////////////
    //                        ENTRYPOINTS                         //
    ////////////////////////////////////////////////////////////////

    function deployFPAC(address _proxyAdmin, address _systemOwnerSafe, address _superchainConfigProxy) public {
        console.log("Deploying a fresh FPAC system and OptimismPortal2 implementation.");

        prankDeployment("ProxyAdmin", msg.sender);
        prankDeployment("SystemOwnerSafe", msg.sender);
        prankDeployment("SuperchainConfigProxy", _superchainConfigProxy);

        // Deploy the proxies.
        deployERC1967Proxy("DisputeGameFactoryProxy");
        deployERC1967Proxy("DelayedWETHProxy");
        deployERC1967Proxy("AnchorStateRegistryProxy");

        // Deploy implementations.
        deployDisputeGameFactory();
        deployDelayedWETH();
        deployAnchorStateRegistry();
        deployPreimageOracle();
        deployMips();

        // Deploy the new `OptimismPortal` implementation.
        deployOptimismPortal2();

        // Initialize the proxies.
        initializeDisputeGameFactoryProxy();
        initializeDelayedWETHProxy();
        initializeAnchorStateRegistryProxy();

        // Deploy the Cannon Fault game implementation and set it as game ID = 0.
        setCannonFaultGameImplementation({ _allowUpgrade: false });
        // Deploy the Permissioned Cannon Fault game implementation and set it as game ID = 1.
        setPermissionedCannonFaultGameImplementation({ _allowUpgrade: false });

        // Transfer ownership of the DisputeGameFactory to the SystemOwnerSafe, and transfer the administrative rights
        // of the DisputeGameFactoryProxy to the ProxyAdmin.
        transferDGFOwnershipFinal({ _proxyAdmin: _proxyAdmin, _systemOwnerSafe: _systemOwnerSafe });
        transferWethOwnershipFinal({ _proxyAdmin: _proxyAdmin, _systemOwnerSafe: _systemOwnerSafe });

        // Run post-deployment assertions.
        postDeployAssertions({ _proxyAdmin: _proxyAdmin, _systemOwnerSafe: _systemOwnerSafe });

        // Print overview
        printConfigReview();
    }

    ////////////////////////////////////////////////////////////////
    //                          HELPERS                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Initializes the DisputeGameFactoryProxy with the DisputeGameFactory.
    function initializeDisputeGameFactoryProxy() internal broadcast {
        console.log("Initializing DisputeGameFactoryProxy with DisputeGameFactory.");

        address dgfProxy = mustGetAddress("DisputeGameFactoryProxy");
        Proxy(payable(dgfProxy)).upgradeToAndCall(
            mustGetAddress("DisputeGameFactory"), abi.encodeCall(DisputeGameFactory.initialize, msg.sender)
        );
    }

    function initializeDelayedWETHProxy() internal broadcast {
        console.log("Initializing DelayedWETHProxy with DelayedWETH.");

        address wethProxy = mustGetAddress("DelayedWETHProxy");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        Proxy(payable(wethProxy)).upgradeToAndCall(
            mustGetAddress("DelayedWETH"),
            abi.encodeCall(DelayedWETH.initialize, (msg.sender, SuperchainConfig(superchainConfigProxy)))
        );
    }

    function initializeAnchorStateRegistryProxy() internal broadcast {
        console.log("Initializing AnchorStateRegistryProxy with AnchorStateRegistry.");

        AnchorStateRegistry.StartingAnchorRoot[] memory roots = new AnchorStateRegistry.StartingAnchorRoot[](3);
        roots[0] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.CANNON,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[1] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.PERMISSIONED_CANNON,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });

        address asrProxy = mustGetAddress("AnchorStateRegistryProxy");
        Proxy(payable(asrProxy)).upgradeToAndCall(
            mustGetAddress("AnchorStateRegistry"), abi.encodeCall(AnchorStateRegistry.initialize, (roots))
        );
    }

    /// @notice Transfers admin rights of the `DisputeGameFactoryProxy` to the `ProxyAdmin` and sets the
    ///         `DisputeGameFactory` owner to the `SystemOwnerSafe`.
    function transferDGFOwnershipFinal(address _proxyAdmin, address _systemOwnerSafe) internal broadcast {
        DisputeGameFactory dgf = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));

        // Transfer the ownership of the DisputeGameFactory to the SystemOwnerSafe.
        dgf.transferOwnership(_systemOwnerSafe);

        // Transfer the admin rights of the DisputeGameFactoryProxy to the ProxyAdmin.
        Proxy prox = Proxy(payable(address(dgf)));
        prox.changeAdmin(_proxyAdmin);
    }

    /// @notice Transfers admin rights of the `DelayedWETHProxy` to the `ProxyAdmin` and sets the
    ///         `DelayedWETH` owner to the `SystemOwnerSafe`.
    function transferWethOwnershipFinal(address _proxyAdmin, address _systemOwnerSafe) internal broadcast {
        DelayedWETH weth = DelayedWETH(mustGetAddress("DelayedWETHProxy"));

        // Transfer the ownership of the DelayedWETH to the SystemOwnerSafe.
        weth.transferOwnership(_systemOwnerSafe);

        // Transfer the admin rights of the DelayedWETHProxy to the ProxyAdmin.
        Proxy prox = Proxy(payable(address(weth)));
        prox.changeAdmin(_proxyAdmin);
    }

    /// @notice Checks that the deployed system is configured correctly.
    function postDeployAssertions(address _proxyAdmin, address _systemOwnerSafe) internal {
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.OptimismPortal2 = mustGetAddress("OptimismPortal2");

        // Ensure that `useFaultProofs` is set to `true`.
        assertTrue(cfg.useFaultProofs());

        // Ensure the contracts are owned by the correct entities.
        address dgfProxyAddr = mustGetAddress("DisputeGameFactoryProxy");
        DisputeGameFactory dgfProxy = DisputeGameFactory(dgfProxyAddr);
        assertEq(address(uint160(uint256(vm.load(dgfProxyAddr, Constants.PROXY_OWNER_ADDRESS)))), _proxyAdmin);
        ChainAssertions.checkDisputeGameFactory(contracts, _systemOwnerSafe);
        address wethProxyAddr = mustGetAddress("DelayedWETHProxy");
        assertEq(address(uint160(uint256(vm.load(wethProxyAddr, Constants.PROXY_OWNER_ADDRESS)))), _proxyAdmin);
        ChainAssertions.checkDelayedWETH(contracts, cfg, true, _systemOwnerSafe);

        // Check the config elements in the deployed contracts.
        ChainAssertions.checkOptimismPortal2(contracts, cfg, false);

        PreimageOracle oracle = PreimageOracle(mustGetAddress("PreimageOracle"));
        assertEq(oracle.minProposalSize(), cfg.preimageOracleMinProposalSize());
        assertEq(oracle.challengePeriod(), cfg.preimageOracleChallengePeriod());

        MIPS mips = MIPS(mustGetAddress("Mips"));
        assertEq(address(mips.oracle()), address(oracle));

        // Check the FaultDisputeGame configuration.
        FaultDisputeGame gameImpl = FaultDisputeGame(payable(address(dgfProxy.gameImpls(GameTypes.CANNON))));
        assertEq(gameImpl.maxGameDepth(), cfg.faultGameMaxDepth());
        assertEq(gameImpl.splitDepth(), cfg.faultGameSplitDepth());
        assertEq(gameImpl.gameDuration().raw(), cfg.faultGameMaxDuration());
        assertEq(gameImpl.absolutePrestate().raw(), bytes32(cfg.faultGameAbsolutePrestate()));

        // Check the security override yoke configuration.
        PermissionedDisputeGame soyGameImpl =
            PermissionedDisputeGame(payable(address(dgfProxy.gameImpls(GameTypes.PERMISSIONED_CANNON))));
        assertEq(soyGameImpl.maxGameDepth(), cfg.faultGameMaxDepth());
        assertEq(soyGameImpl.splitDepth(), cfg.faultGameSplitDepth());
        assertEq(soyGameImpl.gameDuration().raw(), cfg.faultGameMaxDuration());
        assertEq(soyGameImpl.absolutePrestate().raw(), bytes32(cfg.faultGameAbsolutePrestate()));

        // Check the AnchorStateRegistry configuration.
        AnchorStateRegistry asr = AnchorStateRegistry(mustGetAddress("AnchorStateRegistry"));
        (Hash root1, uint256 l2BlockNumber1) = asr.anchors(GameTypes.CANNON);
        (Hash root2, uint256 l2BlockNumber2) = asr.anchors(GameTypes.PERMISSIONED_CANNON);
        assertEq(root1.raw(), cfg.faultGameGenesisOutputRoot());
        assertEq(root2.raw(), cfg.faultGameGenesisOutputRoot());
        assertEq(l2BlockNumber1, cfg.faultGameGenesisBlock());
        assertEq(l2BlockNumber2, cfg.faultGameGenesisBlock());
    }

    /// @notice Prints a review of the fault proof configuration section of the deploy config.
    function printConfigReview() internal view {
        console.log(unicode"📖 FaultDisputeGame Config Overview (chainid: %d)", block.chainid);
        console.log("    0. Use Fault Proofs: %s", cfg.useFaultProofs() ? "true" : "false");
        console.log("    1. Absolute Prestate: %x", cfg.faultGameAbsolutePrestate());
        console.log("    2. Max Depth: %d", cfg.faultGameMaxDepth());
        console.log("    3. Output / Execution split Depth: %d", cfg.faultGameSplitDepth());
        console.log("    4. Game Duration (seconds): %d", cfg.faultGameMaxDuration());
        console.log("    5. L2 Genesis block number: %d", cfg.faultGameGenesisBlock());
        console.log("    6. L2 Genesis output root: %x", uint256(cfg.faultGameGenesisOutputRoot()));
        console.log("    7. Proof Maturity Delay (seconds): ", cfg.proofMaturityDelaySeconds());
        console.log("    8. Dispute Game Finality Delay (seconds): ", cfg.disputeGameFinalityDelaySeconds());
        console.log("    9. Respected Game Type: ", cfg.respectedGameType());
        console.log("   10. Preimage Oracle Min Proposal Size (bytes): ", cfg.preimageOracleMinProposalSize());
        console.log("   11. Preimage Oracle Challenge Period (seconds): ", cfg.preimageOracleChallengePeriod());
    }
}

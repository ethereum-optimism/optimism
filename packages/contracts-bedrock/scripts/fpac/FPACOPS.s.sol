// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Scripts
import { StdAssertions } from "forge-std/StdAssertions.sol";
import "scripts/deploy/Deploy.s.sol";

// Contracts
import { Proxy } from "src/universal/Proxy.sol";

// Libraries
import "src/dispute/lib/Types.sol";

// Interfaces
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";
import { IDelayedWETH } from "src/dispute/interfaces/IDelayedWETH.sol";
import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "src/dispute/interfaces/IPermissionedDisputeGame.sol";

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
        transferAnchorStateOwnershipFinal({ _proxyAdmin: _proxyAdmin });

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
            mustGetAddress("DisputeGameFactory"), abi.encodeCall(IDisputeGameFactory.initialize, msg.sender)
        );

        // Set the initialization bonds for the FaultDisputeGame and PermissionedDisputeGame.
        IDisputeGameFactory dgf = IDisputeGameFactory(dgfProxy);
        dgf.setInitBond(GameTypes.CANNON, 0.08 ether);
        dgf.setInitBond(GameTypes.PERMISSIONED_CANNON, 0.08 ether);
    }

    function initializeDelayedWETHProxy() internal broadcast {
        console.log("Initializing DelayedWETHProxy with DelayedWETH.");

        address wethProxy = mustGetAddress("DelayedWETHProxy");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        Proxy(payable(wethProxy)).upgradeToAndCall(
            mustGetAddress("DelayedWETH"),
            abi.encodeCall(IDelayedWETH.initialize, (msg.sender, ISuperchainConfig(superchainConfigProxy)))
        );
    }

    function initializeAnchorStateRegistryProxy() internal broadcast {
        console.log("Initializing AnchorStateRegistryProxy with AnchorStateRegistry.");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        ISuperchainConfig superchainConfig = ISuperchainConfig(superchainConfigProxy);

        IAnchorStateRegistry.StartingAnchorRoot[] memory roots = new IAnchorStateRegistry.StartingAnchorRoot[](2);
        roots[0] = IAnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.CANNON,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[1] = IAnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.PERMISSIONED_CANNON,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });

        address asrProxy = mustGetAddress("AnchorStateRegistryProxy");
        Proxy(payable(asrProxy)).upgradeToAndCall(
            mustGetAddress("AnchorStateRegistry"),
            abi.encodeCall(IAnchorStateRegistry.initialize, (roots, superchainConfig))
        );
    }

    /// @notice Transfers admin rights of the `DisputeGameFactoryProxy` to the `ProxyAdmin` and sets the
    ///         `DisputeGameFactory` owner to the `SystemOwnerSafe`.
    function transferDGFOwnershipFinal(address _proxyAdmin, address _systemOwnerSafe) internal broadcast {
        IDisputeGameFactory dgf = IDisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));

        // Transfer the ownership of the DisputeGameFactory to the SystemOwnerSafe.
        dgf.transferOwnership(_systemOwnerSafe);

        // Transfer the admin rights of the DisputeGameFactoryProxy to the ProxyAdmin.
        Proxy prox = Proxy(payable(address(dgf)));
        prox.changeAdmin(_proxyAdmin);
    }

    /// @notice Transfers admin rights of the `DelayedWETHProxy` to the `ProxyAdmin` and sets the
    ///         `DelayedWETH` owner to the `SystemOwnerSafe`.
    function transferWethOwnershipFinal(address _proxyAdmin, address _systemOwnerSafe) internal broadcast {
        IDelayedWETH weth = IDelayedWETH(mustGetAddress("DelayedWETHProxy"));

        // Transfer the ownership of the DelayedWETH to the SystemOwnerSafe.
        weth.transferOwnership(_systemOwnerSafe);

        // Transfer the admin rights of the DelayedWETHProxy to the ProxyAdmin.
        Proxy prox = Proxy(payable(address(weth)));
        prox.changeAdmin(_proxyAdmin);
    }

    /// @notice Transfers admin rights of the `AnchorStateRegistryProxy` to the `ProxyAdmin`.
    function transferAnchorStateOwnershipFinal(address _proxyAdmin) internal broadcast {
        IAnchorStateRegistry asr = IAnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy"));

        // Transfer the admin rights of the AnchorStateRegistryProxy to the ProxyAdmin.
        Proxy prox = Proxy(payable(address(asr)));
        prox.changeAdmin(_proxyAdmin);
    }

    /// @notice Checks that the deployed system is configured correctly.
    function postDeployAssertions(address _proxyAdmin, address _systemOwnerSafe) internal view {
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.OptimismPortal2 = mustGetAddress("OptimismPortal2");

        // Ensure that `useFaultProofs` is set to `true`.
        assertTrue(cfg.useFaultProofs());

        // Ensure the contracts are owned by the correct entities.
        address dgfProxyAddr = mustGetAddress("DisputeGameFactoryProxy");
        IDisputeGameFactory dgfProxy = IDisputeGameFactory(dgfProxyAddr);
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

        // Check the AnchorStateRegistry configuration.
        IAnchorStateRegistry asr = IAnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy"));
        (Hash root1, uint256 l2BlockNumber1) = asr.anchors(GameTypes.CANNON);
        (Hash root2, uint256 l2BlockNumber2) = asr.anchors(GameTypes.PERMISSIONED_CANNON);
        assertEq(root1.raw(), cfg.faultGameGenesisOutputRoot());
        assertEq(root2.raw(), cfg.faultGameGenesisOutputRoot());
        assertEq(l2BlockNumber1, cfg.faultGameGenesisBlock());
        assertEq(l2BlockNumber2, cfg.faultGameGenesisBlock());

        // Check the FaultDisputeGame configuration.
        IFaultDisputeGame gameImpl = IFaultDisputeGame(payable(address(dgfProxy.gameImpls(GameTypes.CANNON))));
        assertEq(gameImpl.maxGameDepth(), cfg.faultGameMaxDepth());
        assertEq(gameImpl.splitDepth(), cfg.faultGameSplitDepth());
        assertEq(gameImpl.clockExtension().raw(), cfg.faultGameClockExtension());
        assertEq(gameImpl.maxClockDuration().raw(), cfg.faultGameMaxClockDuration());
        assertEq(gameImpl.absolutePrestate().raw(), bytes32(cfg.faultGameAbsolutePrestate()));
        assertEq(address(gameImpl.weth()), wethProxyAddr);
        assertEq(address(gameImpl.anchorStateRegistry()), address(asr));
        assertEq(address(gameImpl.vm()), address(mips));

        // Check the security override yoke configuration.
        IPermissionedDisputeGame soyGameImpl =
            IPermissionedDisputeGame(payable(address(dgfProxy.gameImpls(GameTypes.PERMISSIONED_CANNON))));
        assertEq(soyGameImpl.proposer(), cfg.l2OutputOracleProposer());
        assertEq(soyGameImpl.challenger(), cfg.l2OutputOracleChallenger());
        assertEq(soyGameImpl.maxGameDepth(), cfg.faultGameMaxDepth());
        assertEq(soyGameImpl.splitDepth(), cfg.faultGameSplitDepth());
        assertEq(soyGameImpl.clockExtension().raw(), cfg.faultGameClockExtension());
        assertEq(soyGameImpl.maxClockDuration().raw(), cfg.faultGameMaxClockDuration());
        assertEq(soyGameImpl.absolutePrestate().raw(), bytes32(cfg.faultGameAbsolutePrestate()));
        assertEq(address(soyGameImpl.weth()), wethProxyAddr);
        assertEq(address(soyGameImpl.anchorStateRegistry()), address(asr));
        assertEq(address(soyGameImpl.vm()), address(mips));
    }

    /// @notice Prints a review of the fault proof configuration section of the deploy config.
    function printConfigReview() internal view {
        console.log(unicode"📖 FaultDisputeGame Config Overview (chainid: %d)", block.chainid);
        console.log("    0. Use Fault Proofs: %s", cfg.useFaultProofs() ? "true" : "false");
        console.log("    1. Absolute Prestate: %x", cfg.faultGameAbsolutePrestate());
        console.log("    2. Max Depth: %d", cfg.faultGameMaxDepth());
        console.log("    3. Output / Execution split Depth: %d", cfg.faultGameSplitDepth());
        console.log("    4. Clock Extension (seconds): %d", cfg.faultGameClockExtension());
        console.log("    5. Max Clock Duration (seconds): %d", cfg.faultGameMaxClockDuration());
        console.log("    6. L2 Genesis block number: %d", cfg.faultGameGenesisBlock());
        console.log("    7. L2 Genesis output root: %x", uint256(cfg.faultGameGenesisOutputRoot()));
        console.log("    8. Proof Maturity Delay (seconds): ", cfg.proofMaturityDelaySeconds());
        console.log("    9. Dispute Game Finality Delay (seconds): ", cfg.disputeGameFinalityDelaySeconds());
        console.log("   10. Respected Game Type: ", cfg.respectedGameType());
        console.log("   11. Preimage Oracle Min Proposal Size (bytes): ", cfg.preimageOracleMinProposalSize());
        console.log("   12. Preimage Oracle Challenge Period (seconds): ", cfg.preimageOracleChallengePeriod());
    }
}

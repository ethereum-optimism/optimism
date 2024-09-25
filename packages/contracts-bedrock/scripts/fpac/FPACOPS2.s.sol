// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Scripts
import { StdAssertions } from "forge-std/StdAssertions.sol";
import "scripts/deploy/Deploy.s.sol";

// Libraries
import "src/dispute/lib/Types.sol";

// Interfaces
import { IProxy } from "src/universal/interfaces/IProxy.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";
import { IDelayedWETH } from "src/dispute/interfaces/IDelayedWETH.sol";
import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "src/dispute/interfaces/IPermissionedDisputeGame.sol";

/// @notice Deploys new implementations of the FaultDisputeGame contract and its dependencies
///         assuming that the DisputeGameFactory contract does not need to be modified. Assumes
///         that the System Owner will update the DisputeGameFactory to point to these new
///         contracts at a later point in time.
contract FPACOPS2 is Deploy, StdAssertions {
    ////////////////////////////////////////////////////////////////
    //                        ENTRYPOINTS                         //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploys an updated FP system with new FaultDisputeGame contracts and new
    ///         DelayedWETH contracts. Deploys a new implementation of the
    ///         AnchorStateRegistry. Does not deploy a new DisputeGameFactory. System
    ///         Owner is responsible for updating implementations later.
    /// @param _proxyAdmin Address of the ProxyAdmin contract to transfer ownership to.
    /// @param _finalSystemOwner Address of the SystemOwner.
    /// @param _superchainConfigProxy Address of the SuperchainConfig proxy contract.
    /// @param _disputeGameFactoryProxy Address of the DisputeGameFactory proxy contract.
    /// @param _anchorStateRegistryProxy Address of the AnchorStateRegistry proxy contract.
    function deployFPAC2(
        address _proxyAdmin,
        address _finalSystemOwner,
        address _superchainConfigProxy,
        address _disputeGameFactoryProxy,
        address _anchorStateRegistryProxy
    )
        public
    {
        console.log("Deploying updated FP contracts.");

        // Prank required deployments.
        prankDeployment("ProxyAdmin", msg.sender);
        prankDeployment("FinalSystemOwner", msg.sender);
        prankDeployment("SuperchainConfigProxy", _superchainConfigProxy);
        prankDeployment("DisputeGameFactoryProxy", _disputeGameFactoryProxy);
        prankDeployment("AnchorStateRegistryProxy", _anchorStateRegistryProxy);

        // Deploy the proxies.
        deployERC1967Proxy("DelayedWETHProxy");
        deployERC1967Proxy("PermissionedDelayedWETHProxy");

        // Deploy implementations.
        deployDelayedWETH();
        deployAnchorStateRegistry();
        deployPreimageOracle();
        deployMips();

        // Initialize the proxies.
        initializeDelayedWETHProxy();
        initializePermissionedDelayedWETHProxy();

        // Deploy the new game implementations.
        deployCannonDisputeGame();
        deployPermissionedDisputeGame();

        // Transfer ownership of DelayedWETH to ProxyAdmin.
        transferWethOwnershipFinal({ _proxyAdmin: _proxyAdmin, _finalSystemOwner: _finalSystemOwner });
        transferPermissionedWETHOwnershipFinal({ _proxyAdmin: _proxyAdmin, _finalSystemOwner: _finalSystemOwner });

        // Run post-deployment assertions.
        postDeployAssertions({ _proxyAdmin: _proxyAdmin, _finalSystemOwner: _finalSystemOwner });

        // Print overview.
        printConfigReview();
    }

    ////////////////////////////////////////////////////////////////
    //                          HELPERS                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploys the standard Cannon version of the FaultDisputeGame.
    function deployCannonDisputeGame() internal broadcast {
        console.log("Deploying CannonFaultDisputeGame implementation");

        DeployUtils.create2AndSave({
            _save: this,
            _name: "FaultDisputeGame",
            _nick: "CannonFaultDisputeGame",
            _args: DeployUtils.encodeConstructor(
                abi.encodeCall(
                    IFaultDisputeGame.__constructor__,
                    (
                        GameTypes.CANNON,
                        loadMipsAbsolutePrestate(),
                        cfg.faultGameMaxDepth(),
                        cfg.faultGameSplitDepth(),
                        Duration.wrap(uint64(cfg.faultGameClockExtension())),
                        Duration.wrap(uint64(cfg.faultGameMaxClockDuration())),
                        IBigStepper(mustGetAddress("Mips")),
                        IDelayedWETH(mustGetAddress("DelayedWETHProxy")),
                        IAnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                        cfg.l2ChainID()
                    )
                )
            ),
            _salt: _implSalt()
        });
    }

    /// @notice Deploys the PermissionedDisputeGame.
    function deployPermissionedDisputeGame() internal broadcast {
        console.log("Deploying PermissionedDisputeGame implementation");

        DeployUtils.create2AndSave({
            _save: this,
            _name: "PermissionedDisputeGame",
            _args: DeployUtils.encodeConstructor(
                abi.encodeCall(
                    IPermissionedDisputeGame.__constructor__,
                    (
                        GameTypes.PERMISSIONED_CANNON,
                        loadMipsAbsolutePrestate(),
                        cfg.faultGameMaxDepth(),
                        cfg.faultGameSplitDepth(),
                        Duration.wrap(uint64(cfg.faultGameClockExtension())),
                        Duration.wrap(uint64(cfg.faultGameMaxClockDuration())),
                        IBigStepper(mustGetAddress("Mips")),
                        IDelayedWETH(mustGetAddress("PermissionedDelayedWETHProxy")),
                        IAnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                        cfg.l2ChainID(),
                        cfg.l2OutputOracleProposer(),
                        cfg.l2OutputOracleChallenger()
                    )
                )
            ),
            _salt: _implSalt()
        });
    }

    /// @notice Initializes the DelayedWETH proxy.
    function initializeDelayedWETHProxy() internal broadcast {
        console.log("Initializing DelayedWETHProxy with DelayedWETH.");

        address wethProxy = mustGetAddress("DelayedWETHProxy");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        IProxy(payable(wethProxy)).upgradeToAndCall(
            mustGetAddress("DelayedWETH"),
            abi.encodeCall(IDelayedWETH.initialize, (msg.sender, ISuperchainConfig(superchainConfigProxy)))
        );
    }

    /// @notice Initializes the permissioned DelayedWETH proxy.
    function initializePermissionedDelayedWETHProxy() internal broadcast {
        console.log("Initializing permissioned DelayedWETHProxy with DelayedWETH.");

        address wethProxy = mustGetAddress("PermissionedDelayedWETHProxy");
        address superchainConfigProxy = mustGetAddress("SuperchainConfigProxy");
        IProxy(payable(wethProxy)).upgradeToAndCall(
            mustGetAddress("DelayedWETH"),
            abi.encodeCall(IDelayedWETH.initialize, (msg.sender, ISuperchainConfig(superchainConfigProxy)))
        );
    }

    /// @notice Transfers admin rights of the `DelayedWETHProxy` to the `ProxyAdmin` and sets the
    ///         `DelayedWETH` owner to the `FinalSystemOwner`.
    function transferWethOwnershipFinal(address _proxyAdmin, address _finalSystemOwner) internal broadcast {
        console.log("Transferring ownership of DelayedWETHProxy");

        IDelayedWETH weth = IDelayedWETH(mustGetAddress("DelayedWETHProxy"));

        // Transfer the ownership of the DelayedWETH to the FinalSystemOwner.
        weth.transferOwnership(_finalSystemOwner);

        // Transfer the admin rights of the DelayedWETHProxy to the ProxyAdmin.
        IProxy prox = IProxy(payable(address(weth)));
        prox.changeAdmin(_proxyAdmin);
    }

    /// @notice Transfers admin rights of the permissioned `DelayedWETHProxy` to the `ProxyAdmin`
    ///         and sets the `DelayedWETH` owner to the `FinalSystemOwner`.
    function transferPermissionedWETHOwnershipFinal(
        address _proxyAdmin,
        address _finalSystemOwner
    )
        internal
        broadcast
    {
        console.log("Transferring ownership of permissioned DelayedWETHProxy");

        IDelayedWETH weth = IDelayedWETH(mustGetAddress("PermissionedDelayedWETHProxy"));

        // Transfer the ownership of the DelayedWETH to the FinalSystemOwner.
        weth.transferOwnership(_finalSystemOwner);

        // Transfer the admin rights of the DelayedWETHProxy to the ProxyAdmin.
        IProxy prox = IProxy(payable(address(weth)));
        prox.changeAdmin(_proxyAdmin);
    }

    /// @notice Checks that the deployed system is configured correctly.
    function postDeployAssertions(address _proxyAdmin, address _finalSystemOwner) internal view {
        Types.ContractSet memory contracts = _proxiesUnstrict();

        // Ensure that `useFaultProofs` is set to `true`.
        assertTrue(cfg.useFaultProofs());

        // Verify that the DGF is owned by the ProxyAdmin.
        address dgfProxyAddr = mustGetAddress("DisputeGameFactoryProxy");
        assertEq(address(uint160(uint256(vm.load(dgfProxyAddr, Constants.PROXY_OWNER_ADDRESS)))), _proxyAdmin);

        // Verify that DelayedWETH is owned by the ProxyAdmin.
        address wethProxyAddr = mustGetAddress("DelayedWETHProxy");
        assertEq(address(uint160(uint256(vm.load(wethProxyAddr, Constants.PROXY_OWNER_ADDRESS)))), _proxyAdmin);

        // Verify that permissioned DelayedWETH is owned by the ProxyAdmin.
        address soyWethProxyAddr = mustGetAddress("PermissionedDelayedWETHProxy");
        assertEq(address(uint160(uint256(vm.load(soyWethProxyAddr, Constants.PROXY_OWNER_ADDRESS)))), _proxyAdmin);

        // Run standard assertions for DGF and DelayedWETH.
        ChainAssertions.checkDisputeGameFactory(contracts, _finalSystemOwner);
        ChainAssertions.checkDelayedWETH(contracts, cfg, true, _finalSystemOwner);
        ChainAssertions.checkPermissionedDelayedWETH(contracts, cfg, true, _finalSystemOwner);

        // Verify PreimageOracle configuration.
        IPreimageOracle oracle = IPreimageOracle(mustGetAddress("PreimageOracle"));
        assertEq(oracle.minProposalSize(), cfg.preimageOracleMinProposalSize());
        assertEq(oracle.challengePeriod(), cfg.preimageOracleChallengePeriod());

        // Verify MIPS configuration.
        IMIPS mips = IMIPS(mustGetAddress("Mips"));
        assertEq(address(mips.oracle()), address(oracle));

        // Grab ASR
        IAnchorStateRegistry asr = IAnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy"));

        // Verify FaultDisputeGame configuration.
        address gameAddr = mustGetAddress("CannonFaultDisputeGame");
        IFaultDisputeGame gameImpl = IFaultDisputeGame(payable(gameAddr));
        assertEq(gameImpl.maxGameDepth(), cfg.faultGameMaxDepth());
        assertEq(gameImpl.splitDepth(), cfg.faultGameSplitDepth());
        assertEq(gameImpl.clockExtension().raw(), cfg.faultGameClockExtension());
        assertEq(gameImpl.maxClockDuration().raw(), cfg.faultGameMaxClockDuration());
        assertEq(gameImpl.absolutePrestate().raw(), bytes32(cfg.faultGameAbsolutePrestate()));
        assertEq(address(gameImpl.weth()), wethProxyAddr);
        assertEq(address(gameImpl.anchorStateRegistry()), address(asr));
        assertEq(address(gameImpl.vm()), address(mips));

        // Verify security override yoke configuration.
        address soyGameAddr = mustGetAddress("PermissionedDisputeGame");
        IPermissionedDisputeGame soyGameImpl = IPermissionedDisputeGame(payable(soyGameAddr));
        assertEq(soyGameImpl.proposer(), cfg.l2OutputOracleProposer());
        assertEq(soyGameImpl.challenger(), cfg.l2OutputOracleChallenger());
        assertEq(soyGameImpl.maxGameDepth(), cfg.faultGameMaxDepth());
        assertEq(soyGameImpl.splitDepth(), cfg.faultGameSplitDepth());
        assertEq(soyGameImpl.clockExtension().raw(), cfg.faultGameClockExtension());
        assertEq(soyGameImpl.maxClockDuration().raw(), cfg.faultGameMaxClockDuration());
        assertEq(soyGameImpl.absolutePrestate().raw(), bytes32(cfg.faultGameAbsolutePrestate()));
        assertEq(address(soyGameImpl.weth()), soyWethProxyAddr);
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

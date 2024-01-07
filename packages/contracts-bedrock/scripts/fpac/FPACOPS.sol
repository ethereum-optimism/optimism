// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Proxy } from "src/universal/Proxy.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { StdAssertions } from "forge-std/StdAssertions.sol";
import "scripts/Deploy.s.sol";

/// @notice Deploys the Fault Proof Alpha Chad contracts. These contracts are currently in-development, and the system
///         is independent of the live protocol. Once the contracts are integrated with the live protocol, this script
///         should be deleted.
contract FPACOPS is Deploy, StdAssertions {
    ////////////////////////////////////////////////////////////////
    //                        ENTRYPOINTS                         //
    ////////////////////////////////////////////////////////////////

    function deployFPAC() public {
        console.log("Deploying a fresh FPAC system");

        // Mock the proxy admin & system owner safe so that the DisputeGameFactory & proxy are governed by the deployer.
        prankDeployment("ProxyAdmin", msg.sender);
        prankDeployment("SystemOwnerSafe", msg.sender);

        // Deploy the DisputeGameFactoryProxy.
        deployERC1967Proxy("DisputeGameFactoryProxy");

        // Deploy implementations.
        deployDisputeGameFactory();
        deployPreimageOracle();
        deployMips();

        // Initialize the DisputeGameFactoryProxy.
        initializeDisputeGameFactoryProxy();

        // Deploy the Cannon Fault game implementation and set it as game ID = 0.
        setCannonFaultGameImplementation({ _allowUpgrade: false });

        // Ensure `msg.sender` owns the system.
        postDeployAssertions();

        // Print overview
        printConfigReview();
    }

    function upgradeGameImpl(address _dgf, address _mips) public {
        prankDeployment("DisputeGameFactoryProxy", _dgf);
        prankDeployment("Mips", _mips);

        setCannonFaultGameImplementation({ _allowUpgrade: true });
    }

    function updateInitBond(address _dgf, GameType _gameType, uint256 _newBond) public {
        vm.startBroadcast(msg.sender);
        DisputeGameFactory dgfProxy = DisputeGameFactory(_dgf);
        dgfProxy.setInitBond(_gameType, _newBond);
        vm.stopBroadcast();
    }

    ////////////////////////////////////////////////////////////////
    //                          HELPERS                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Initializes the DisputeGameFactoryProxy with the DisputeGameFactory.
    function initializeDisputeGameFactoryProxy() internal onlyTestnetOrDevnet broadcast {
        address dgfProxy = mustGetAddress("DisputeGameFactoryProxy");
        Proxy(payable(dgfProxy)).upgradeToAndCall(
            mustGetAddress("DisputeGameFactory"), abi.encodeWithSignature("initialize(address)", msg.sender)
        );
    }

    /// @notice Checks that the deployed system is configured correctly.
    function postDeployAssertions() internal {
        // Ensure `msg.sender` owns the deployed system.
        address proxyAdmin = mustGetAddress("ProxyAdmin");
        address dgfProxyAddr = mustGetAddress("DisputeGameFactoryProxy");
        DisputeGameFactory dgfProxy = DisputeGameFactory(dgfProxyAddr);
        assertEq(dgfProxy.owner(), proxyAdmin);
        assertEq(address(uint160(uint256(vm.load(dgfProxyAddr, Constants.PROXY_OWNER_ADDRESS)))), proxyAdmin);

        // Check the config elements.
        FaultDisputeGame gameImpl = FaultDisputeGame(address(dgfProxy.gameImpls(GameTypes.CANNON)));
        assertEq(gameImpl.maxGameDepth(), cfg.faultGameMaxDepth());
        assertEq(gameImpl.splitDepth(), cfg.faultGameSplitDepth());
        assertEq(gameImpl.gameDuration().raw(), cfg.faultGameMaxDuration());
        assertEq(gameImpl.absolutePrestate().raw(), bytes32(cfg.faultGameAbsolutePrestate()));
        assertEq(gameImpl.genesisBlockNumber(), cfg.faultGameGenesisBlock());
        assertEq(gameImpl.genesisOutputRoot().raw(), cfg.faultGameGenesisOutputRoot());
    }

    /// @notice Prints a review of the fault proof configuration section of the deploy config.
    function printConfigReview() internal view {
        console.log(unicode"📖 FaultDisputeGame Config Overview (chainid: %d)", block.chainid);
        console.log("    1. Absolute Prestate: %x", cfg.faultGameAbsolutePrestate());
        console.log("    2. Max Depth: %d", cfg.faultGameMaxDepth());
        console.log("    3. Output / Execution split Depth: %d", cfg.faultGameSplitDepth());
        console.log("    4. Game Duration (seconds): %d", cfg.faultGameMaxDuration());
        console.log("    5. L2 Genesis block number: %d", cfg.faultGameGenesisBlock());
        console.log("    6. L2 Genesis output root: %x", uint256(cfg.faultGameGenesisOutputRoot()));
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Forge
import { Script } from "forge-std/Script.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { GameType, Duration, Claim } from "src/dispute/lib/Types.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";

/// @title AddGameType
contract AddGameType is Script {
    struct Input {
        // Address that will be used for the DummyCaller contract
        address prank;
        // OPCM contract address
        IOPContractsManager opcmImpl;
        // SystemConfig contract address
        ISystemConfig systemConfigProxy;
        // DelayedWETH contract address (optional)
        IDelayedWETH delayedWETHProxy;
        // Game type to add
        GameType disputeGameType;
        // Absolute prestate for the game
        Claim disputeAbsolutePrestate;
        // Maximum game depth
        uint256 disputeMaxGameDepth;
        // Split depth for the game
        uint256 disputeSplitDepth;
        // Clock extension duration
        Duration disputeClockExtension;
        // Maximum clock duration
        Duration disputeMaxClockDuration;
        // Initial bond amount
        uint256 initialBond;
        // VM contract address
        IBigStepper vm;
        // Whether this is a permissioned game
        bool permissioned;
        // Salt mixer for deterministic addresses
        string saltMixer;
    }

    struct Output {
        IDelayedWETH delayedWETHProxy;
        IFaultDisputeGame faultDisputeGameProxy;
    }

    function run(Input memory _agi) public returns (Output memory) {
        // Check if OPCM v2 should be used
        bool useV2 = isDevFeatureOpcmV2Enabled(address(_agi.opcmImpl));

        // Etch DummyCaller contract
        address prank = _agi.prank;

        bytes memory code = vm.getDeployedCode("AddGameType.s.sol:DummyCaller");
        vm.etch(prank, code);
        vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(_agi.opcmImpl)))));
        vm.label(prank, "DummyCaller");

        if (useV2) {
            // V2 path: Use upgrade() with updated game configs
            return addGameTypeWithV2(_agi, prank);
        } else {
            // V1 path: Use dedicated addGameType() function
            return addGameTypeWithV1(_agi, prank);
        }
    }

    /// @notice Check if OPCM v2 should be used based on dev feature flag
    function isDevFeatureOpcmV2Enabled(address _opcmAddr) internal view returns (bool) {
        // Both v1 and v2 share the same interface for this function.
        return IOPContractsManager(_opcmAddr).isDevFeatureEnabled(DevFeatures.OPCM_V2);
    }

    /// @notice Add game type using OPCMv1's addGameType function
    function addGameTypeWithV1(Input memory _agi, address _prank) internal returns (Output memory) {
        // Create the game input
        IOPContractsManager.AddGameInput[] memory gameConfigs = new IOPContractsManager.AddGameInput[](1);
        gameConfigs[0] = IOPContractsManager.AddGameInput({
            saltMixer: _agi.saltMixer,
            systemConfig: _agi.systemConfigProxy,
            delayedWETH: _agi.delayedWETHProxy,
            disputeGameType: _agi.disputeGameType,
            disputeAbsolutePrestate: _agi.disputeAbsolutePrestate,
            disputeMaxGameDepth: _agi.disputeMaxGameDepth,
            disputeSplitDepth: _agi.disputeSplitDepth,
            disputeClockExtension: _agi.disputeClockExtension,
            disputeMaxClockDuration: _agi.disputeMaxClockDuration,
            initialBond: _agi.initialBond,
            vm: _agi.vm,
            permissioned: _agi.permissioned
        });

        // Call into the DummyCaller to perform the delegatecall
        vm.broadcast(msg.sender);

        (bool success, bytes memory result) = DummyCaller(_prank).addGameType(gameConfigs);
        require(success, "AddGameType: addGameType failed");

        // Decode the result and set it in the output
        IOPContractsManager.AddGameOutput[] memory outputs = abi.decode(result, (IOPContractsManager.AddGameOutput[]));
        require(outputs.length == 1, "AddGameType: unexpected number of outputs");
        return Output({ delayedWETHProxy: outputs[0].delayedWETH, faultDisputeGameProxy: outputs[0].faultDisputeGame });
    }

    /// @notice Add game type using OPCMv2's upgrade function
    /// @dev v2 doesn't have a separate addGameType function - we call upgrade() with updated game configs
    function addGameTypeWithV2(Input memory _agi, address _prank) internal returns (Output memory) {
        // Fetch existing game configs from the chain to avoid disabling them
        IOPContractsManagerV2.DisputeGameConfig[] memory existingGames = fetchExistingGameConfigs(address(_agi.systemConfigProxy));

        // Build the new game config
        IOPContractsManagerV2.DisputeGameConfig memory newGame = buildNewGameConfig(_agi);

        // Combine existing and new game configs
        IOPContractsManagerV2.DisputeGameConfig[] memory allGames = new IOPContractsManagerV2.DisputeGameConfig[](existingGames.length + 1);
        for (uint256 i = 0; i < existingGames.length; i++) {
            allGames[i] = existingGames[i];
        }
        allGames[existingGames.length] = newGame;

        // Build the upgrade input
        IOPContractsManagerV2.UpgradeInput memory upgradeInput = IOPContractsManagerV2.UpgradeInput({
            systemConfig: _agi.systemConfigProxy,
            disputeGameConfigs: allGames,
            extraInstructions: new IOPContractsManagerV2.ExtraInstruction[](0)
        });

        // Call into the DummyCallerV2 to perform the delegatecall
        vm.broadcast(msg.sender);
        (bool success, bytes memory result) = DummyCallerV2(_prank).upgrade(upgradeInput);
        require(success, "AddGameType: v2 upgrade failed");

        // Decode the result
        IOPContractsManagerV2.ChainContracts memory chainContracts = abi.decode(result, (IOPContractsManagerV2.ChainContracts));

        // Return output in v1 format for compatibility
        return Output({
            delayedWETHProxy: chainContracts.delayedWETH,
            faultDisputeGameProxy: IFaultDisputeGame(address(chainContracts.disputeGameFactory))
        });
    }

    /// @notice Fetch existing dispute game configs from the chain
    /// @dev Critical for v2 to avoid accidentally disabling existing games
    function fetchExistingGameConfigs(address _systemConfig)
        internal
        view
        returns (IOPContractsManagerV2.DisputeGameConfig[] memory)
    {
        ISystemConfig systemConfig = ISystemConfig(_systemConfig);
        IDisputeGameFactory factory = IDisputeGameFactory(address(systemConfig.disputeGameFactory()));

        // First pass: count how many game types are configured
        uint256 configuredCount = 0;
        uint256 maxGameType = 15; // Check game types 0-15 (reasonable upper bound)

        for (uint256 i = 0; i <= maxGameType; i++) {
            GameType gameType = GameType.wrap(uint32(i));
            IDisputeGame impl = factory.gameImpls(gameType);
            if (address(impl) != address(0)) {
                configuredCount++;
            }
        }

        // Second pass: build the config array
        IOPContractsManagerV2.DisputeGameConfig[] memory configs =
            new IOPContractsManagerV2.DisputeGameConfig[](configuredCount);

        uint256 index = 0;
        for (uint256 i = 0; i <= maxGameType; i++) {
            GameType gameType = GameType.wrap(uint32(i));
            IDisputeGame impl = factory.gameImpls(gameType);

            if (address(impl) != address(0)) {
                // Game type is configured - fetch its parameters
                uint256 initBond = factory.initBonds(gameType);
                bytes memory gameArgs = factory.gameArgs(gameType);

                configs[index] = IOPContractsManagerV2.DisputeGameConfig({
                    enabled: true,
                    initBond: initBond,
                    gameType: gameType,
                    gameArgs: gameArgs
                });
                index++;
            }
        }

        return configs;
    }

    /// @notice Build a new game config from the input
    function buildNewGameConfig(Input memory _agi) internal view returns (IOPContractsManagerV2.DisputeGameConfig memory) {
        bytes memory gameArgs;

        if (_agi.permissioned) {
            // For permissioned games, fetch proposer/challenger from existing PERMISSIONED_CANNON game
            (address proposer, address challenger) = fetchProposerAndChallenger(_agi.systemConfigProxy);

            gameArgs = abi.encode(
                IOPContractsManagerV2.PermissionedDisputeGameConfig({
                    absolutePrestate: _agi.disputeAbsolutePrestate,
                    proposer: proposer,
                    challenger: challenger
                })
            );
        } else {
            // For permissionless games, just encode the absolute prestate
            gameArgs = abi.encode(
                IOPContractsManagerV2.FaultDisputeGameConfig({
                    absolutePrestate: _agi.disputeAbsolutePrestate
                })
            );
        }

        return IOPContractsManagerV2.DisputeGameConfig({
            enabled: true,
            initBond: _agi.initialBond,
            gameType: _agi.disputeGameType,
            gameArgs: gameArgs
        });
    }

    /// @notice Fetch proposer and challenger addresses from existing PERMISSIONED_CANNON game
    function fetchProposerAndChallenger(ISystemConfig _systemConfig)
        internal
        view
        returns (address proposer_, address challenger_)
    {
        IDisputeGameFactory factory = IDisputeGameFactory(address(_systemConfig.disputeGameFactory()));

        // Get the PERMISSIONED_CANNON game implementation (type 1)
        IDisputeGame permissionedGameImpl = factory.gameImpls(GameTypes.PERMISSIONED_CANNON);

        require(
            address(permissionedGameImpl) != address(0),
            "AddGameType: PERMISSIONED_CANNON not found - cannot derive proposer/challenger"
        );

        // Cast to IPermissionedDisputeGame to access proposer() and challenger()
        IPermissionedDisputeGame permissionedGame = IPermissionedDisputeGame(address(permissionedGameImpl));

        proposer_ = permissionedGame.proposer();
        challenger_ = permissionedGame.challenger();

        require(proposer_ != address(0), "AddGameType: proposer address is zero");
        require(challenger_ != address(0), "AddGameType: challenger address is zero");
    }

    function checkOutput(Output memory _ago) internal view {
        DeployUtils.assertValidContractAddress(address(_ago.delayedWETHProxy));
        DeployUtils.assertValidContractAddress(address(_ago.faultDisputeGameProxy));
    }
}

/// @title DummyCaller
/// @notice This contract is used to mimic the contract that is used as the source of the delegatecall to the OPCM.
/// @dev This contract is used for OPCM versions 4.1.0 and above (v1).

contract DummyCaller {
    address internal _opcmAddr;

    function addGameType(IOPContractsManager.AddGameInput[] memory _gameConfigs)
        external
        returns (bool, bytes memory)
    {
        bytes memory data = abi.encodeCall(DummyCaller.addGameType, _gameConfigs);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}

/// @title DummyCallerV2
/// @notice This contract is used for OPCMv2 delegatecalls
/// @dev Uses the upgrade() function instead of addGameType()

contract DummyCallerV2 {
    address internal _opcmAddr;

    function upgrade(IOPContractsManagerV2.UpgradeInput memory _upgradeInput) external returns (bool, bytes memory) {
        bytes memory data = abi.encodeCall(DummyCallerV2.upgrade, _upgradeInput);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { GameTypes, Claim } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";

/// @title SuperGameTestInit
/// @notice Shared base for tests that need super game mode (SUPER_PERMISSIONED_CANNON + SUPER_CANNON_KONA).
///         Provides common state variables and the upgrade helper used by both MigrationValidator
///         and StandardValidator super mode test suites.
abstract contract SuperGameTestInit is CommonTest {
    /// @notice The cannon prestate (used by SUPER_PERMISSIONED_CANNON).
    Claim cannonPrestate;

    /// @notice The cannonKona prestate (used by SUPER_CANNON_KONA).
    Claim cannonKonaPrestate = Claim.wrap(bytes32(keccak256("cannonKonaPrestate")));

    /// @notice The proposer role.
    address proposer;

    /// @notice The challenger role.
    address challenger;

    /// @notice Runs an upgrade that enables SUPER_CANNON_KONA alongside SUPER_PERMISSIONED_CANNON.
    function _enableSuperCannonKona() internal virtual {
        address owner = proxyAdmin.owner();

        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](6);

        // Legacy types (all disabled).
        disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON,
            gameArgs: hex""
        });
        disputeGameConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.PERMISSIONED_CANNON,
            gameArgs: hex""
        });
        disputeGameConfigs[2] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON_KONA,
            gameArgs: hex""
        });

        // Super types (enabled).
        disputeGameConfigs[3] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 0.08 ether,
            gameType: GameTypes.SUPER_PERMISSIONED_CANNON,
            gameArgs: abi.encode(
                IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                    absolutePrestate: cannonPrestate,
                    proposer: proposer,
                    challenger: challenger
                })
            )
        });
        disputeGameConfigs[4] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 0.08 ether,
            gameType: GameTypes.SUPER_CANNON_KONA,
            gameArgs: abi.encode(IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonKonaPrestate }))
        });
        disputeGameConfigs[5] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.ZK_DISPUTE_GAME,
            gameArgs: hex""
        });

        IOPContractsManagerUtils.ExtraInstruction[] memory extraInstructions =
            new IOPContractsManagerUtils.ExtraInstruction[](1);
        extraInstructions[0] = IOPContractsManagerUtils.ExtraInstruction({
            key: "overrides.cfg.startingRespectedGameType",
            data: abi.encode(GameTypes.SUPER_PERMISSIONED_CANNON)
        });

        prankDelegateCall(owner);
        (bool success,) = address(opcmV2).delegatecall(
            abi.encodeCall(
                IOPContractsManagerV2.upgrade,
                (
                    IOPContractsManagerV2.UpgradeInput({
                        systemConfig: systemConfig,
                        disputeGameConfigs: disputeGameConfigs,
                        extraInstructions: extraInstructions
                    })
                )
            )
        );
        assertTrue(success, "super mode upgrade failed");
    }
}

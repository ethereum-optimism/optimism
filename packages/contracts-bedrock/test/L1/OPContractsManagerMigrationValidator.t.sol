// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { GameTypes, Claim } from "src/dispute/lib/Types.sol";
import { Config } from "scripts/libraries/Config.sol";

// Interfaces
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IOPContractsManagerMigrationValidator } from "interfaces/L1/IOPContractsManagerMigrationValidator.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";

/// @title OPContractsManagerMigrationValidator_TestInit
/// @notice Base contract for MigrationValidator tests. Requires SUPER_ROOT_GAMES_MIGRATION flag.
///         Sets up a chain in super mode with both SUPER_PERMISSIONED_CANNON and SUPER_CANNON_KONA,
///         then configures a mock per-chain DGF (empty) so migration validation passes.
abstract contract OPContractsManagerMigrationValidator_TestInit is CommonTest {
    /// @notice The cannon prestate (used by SUPER_PERMISSIONED_CANNON).
    Claim cannonPrestate;

    /// @notice The cannonKona prestate (used by SUPER_CANNON_KONA).
    Claim cannonKonaPrestate = Claim.wrap(bytes32(keccak256("cannonKonaPrestate")));

    /// @notice The proposer role.
    address proposer;

    /// @notice The challenger role.
    address challenger;

    /// @notice The shared DGF (the real DGF from the deploy, which has super games registered).
    IDisputeGameFactory sharedDGF;

    /// @notice The mock per-chain DGF address (returns address(0) for all gameImpls).
    address mockPerChainDGF;

    /// @notice The MigrationValidator instance.
    IOPContractsManagerMigrationValidator migrationValidator;

    function setUp() public virtual override {
        if (!Config.devFeatureSuperRootGamesMigration()) {
            vm.skip(true, "Skipping: requires SUPER_ROOT_GAMES_MIGRATION");
        }
        super.setUp();

        sharedDGF = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));
        migrationValidator = opcmV2.opcmStandardValidator().migrationValidator();

        cannonPrestate = Claim.wrap(bytes32(deploy.cfg().faultGameAbsolutePrestate()));
        proposer = deploy.cfg().l2OutputOracleProposer();
        challenger = deploy.cfg().l2OutputOracleChallenger();

        // Enable SUPER_CANNON_KONA alongside SUPER_PERMISSIONED_CANNON.
        _enableSuperCannonKona();

        // Set up a mock per-chain DGF that returns address(0) for all game types.
        // This simulates a post-migration per-chain DGF with all games cleared.
        mockPerChainDGF = address(0xdead0001);
        _mockEmptyDGF(mockPerChainDGF);

        // Mock systemConfig.disputeGameFactory() to return our empty per-chain DGF.
        vm.mockCall(
            address(systemConfig), abi.encodeCall(ISystemConfig.disputeGameFactory, ()), abi.encode(mockPerChainDGF)
        );

        // Mock portal.ethLockbox() to return the test ETHLockbox.
        // In the real migration the portal would be initialized with the lockbox,
        // but the test deploy doesn't set it.
        address portalAddr = systemConfig.optimismPortal();
        vm.mockCall(portalAddr, abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(address(ethLockbox)));

        // Mock lockbox.authorizedPortals(portal) to return true.
        vm.mockCall(
            address(ethLockbox),
            abi.encodeCall(IETHLockbox.authorizedPortals, (IOptimismPortal2(payable(portalAddr)))),
            abi.encode(true)
        );
    }

    /// @notice Mocks a DGF address to return address(0) for all game type impls.
    function _mockEmptyDGF(address _dgf) internal {
        vm.mockCall(_dgf, abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)), abi.encode(address(0)));
        vm.mockCall(
            _dgf, abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.PERMISSIONED_CANNON)), abi.encode(address(0))
        );
        vm.mockCall(
            _dgf, abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON_KONA)), abi.encode(address(0))
        );
        vm.mockCall(
            _dgf, abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON)), abi.encode(address(0))
        );
        vm.mockCall(
            _dgf,
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(address(0))
        );
        vm.mockCall(
            _dgf, abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON_KONA)), abi.encode(address(0))
        );
    }

    /// @notice Runs an upgrade that enables SUPER_CANNON_KONA alongside SUPER_PERMISSIONED_CANNON.
    function _enableSuperCannonKona() internal {
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
        disputeGameConfigs[3] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.SUPER_CANNON,
            gameArgs: hex""
        });

        // Super types (enabled).
        disputeGameConfigs[4] = IOPContractsManagerUtils.DisputeGameConfig({
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
        disputeGameConfigs[5] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 0.08 ether,
            gameType: GameTypes.SUPER_CANNON_KONA,
            gameArgs: abi.encode(IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonKonaPrestate }))
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

    /// @notice Builds the MigrationValidationInput and calls validateMigration.
    function _validateMigration(bool _allowFailure) internal view returns (string memory) {
        ISystemConfig[] memory chains = new ISystemConfig[](1);
        chains[0] = systemConfig;
        return migrationValidator.validateMigration(
            IOPContractsManagerMigrationValidator.MigrationValidationInput({
                dgf: sharedDGF,
                chainSystemConfigs: chains,
                cannonPrestate: cannonPrestate.raw(),
                cannonKonaPrestate: cannonKonaPrestate.raw(),
                proposer: proposer,
                challenger: challenger
            }),
            _allowFailure
        );
    }
}

/// @title OPContractsManagerMigrationValidator_ValidateMigration_Test
/// @notice Tests that full migration validation passes with correct setup.
contract OPContractsManagerMigrationValidator_ValidateMigration_Test is
    OPContractsManagerMigrationValidator_TestInit
{
    /// @notice Tests that validateMigration succeeds with no errors.
    function test_validateMigration_succeeds() public view {
        string memory errors = _validateMigration(false);
        assertEq(errors, "");
    }

    /// @notice Tests that validateMigration with allowFailure=true also returns empty.
    function test_validateMigration_allowFailureTrue_succeeds() public view {
        string memory errors = _validateMigration(true);
        assertEq(errors, "");
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Scripts
import { ReadImplementationAddresses } from "scripts/deploy/ReadImplementationAddresses.s.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";

/// @title ReadImplementationAddressesTest
/// @notice Tests that ReadImplementationAddresses correctly reads implementation addresses
///         from the deployed contracts. Uses CommonTest to get real deployed contracts.
contract ReadImplementationAddressesTest is CommonTest {
    ReadImplementationAddresses script;

    function setUp() public override {
        super.setUp();
        script = new ReadImplementationAddresses();
    }

    /// @notice Returns the OPCM instance, handling V1 vs V2 feature flag.
    function _opcm() internal view returns (IOPContractsManager) {
        return isDevFeatureEnabled(DevFeatures.OPCM_V2) ? IOPContractsManager(address(opcmV2)) : opcm;
    }

    /// @notice Builds the input struct from the deployed contracts.
    function _buildInput() internal view returns (ReadImplementationAddresses.Input memory input_) {
        input_.addressManager = address(addressManager);
        input_.l1ERC721BridgeProxy = address(l1ERC721Bridge);
        input_.systemConfigProxy = address(systemConfig);
        input_.optimismMintableERC20FactoryProxy = address(l1OptimismMintableERC20Factory);
        input_.l1StandardBridgeProxy = address(l1StandardBridge);
        input_.optimismPortalProxy = address(optimismPortal2);
        input_.disputeGameFactoryProxy = address(disputeGameFactory);
        input_.opcm = address(_opcm());
    }

    /// @notice Tests that ReadImplementationAddresses.run succeeds and returns correct addresses.
    function test_run_succeeds() public {
        ReadImplementationAddresses.Input memory input = _buildInput();
        ReadImplementationAddresses.Output memory output = script.run(input);

        // Get expected implementations from OPCM
        IOPContractsManager opcm_ = _opcm();
        IOPContractsManager.Implementations memory impls = opcm_.implementations();

        // Assert implementations from OPCM match output
        assertEq(output.delayedWETH, impls.delayedWETHImpl, "DelayedWETH should match");
        assertEq(output.anchorStateRegistry, impls.anchorStateRegistryImpl, "AnchorStateRegistry should match");
        assertEq(output.mipsSingleton, impls.mipsImpl, "MIPS singleton should match");
        assertEq(output.faultDisputeGame, impls.faultDisputeGameImpl, "FaultDisputeGame should match");
        assertEq(
            output.permissionedDisputeGame, impls.permissionedDisputeGameImpl, "PermissionedDisputeGame should match"
        );

        // Assert PreimageOracle is read from MIPS
        IMIPS64 mips_ = IMIPS64(impls.mipsImpl);
        assertEq(output.preimageOracleSingleton, address(mips_.oracle()), "PreimageOracle should match");

        // Assert OPCM standard validator
        assertEq(
            output.opcmStandardValidator, address(opcm_.opcmStandardValidator()), "OPCM StandardValidator should match"
        );

        // Assert V1 vs V2 specific fields
        if (isDevFeatureEnabled(DevFeatures.OPCM_V2)) {
            // V2: deployer/upgrader/gameTypeAdder are zero, migrator comes from opcmMigrator()
            assertEq(output.opcmDeployer, address(0), "OPCM Deployer should be zero in V2");
            assertEq(output.opcmUpgrader, address(0), "OPCM Upgrader should be zero in V2");
            assertEq(output.opcmGameTypeAdder, address(0), "OPCM GameTypeAdder should be zero in V2");
            assertEq(
                output.opcmInteropMigrator,
                address(IOPContractsManagerV2(address(opcm_)).opcmMigrator()),
                "OPCM InteropMigrator should match"
            );
        } else {
            // V1: all component addresses come from opcm getters
            assertEq(output.opcmDeployer, address(opcm_.opcmDeployer()), "OPCM Deployer should match");
            assertEq(output.opcmUpgrader, address(opcm_.opcmUpgrader()), "OPCM Upgrader should match");
            assertEq(output.opcmGameTypeAdder, address(opcm_.opcmGameTypeAdder()), "OPCM GameTypeAdder should match");
            assertEq(
                output.opcmInteropMigrator, address(opcm_.opcmInteropMigrator()), "OPCM InteropMigrator should match"
            );
        }
    }

    /// @notice Tests that ReadImplementationAddresses.runWithBytes succeeds.
    function test_runWithBytes_succeeds() public {
        ReadImplementationAddresses.Input memory input = _buildInput();
        bytes memory inputBytes = abi.encode(input);

        bytes memory outputBytes = script.runWithBytes(inputBytes);
        ReadImplementationAddresses.Output memory output = abi.decode(outputBytes, (ReadImplementationAddresses.Output));

        // Get expected implementations from OPCM
        IOPContractsManager opcm_ = _opcm();
        IOPContractsManager.Implementations memory impls = opcm_.implementations();

        // Assert key values match
        assertEq(output.delayedWETH, impls.delayedWETHImpl, "DelayedWETH should match");
        assertEq(output.mipsSingleton, impls.mipsImpl, "MIPS singleton should match");
        assertEq(
            output.opcmStandardValidator, address(opcm_.opcmStandardValidator()), "OPCM StandardValidator should match"
        );
    }

    /// @notice Tests that the script reverts when OPCM address has no code.
    function test_run_opcmCodeLengthZero_reverts() public {
        ReadImplementationAddresses.Input memory input = _buildInput();
        input.opcm = address(0);

        vm.expectRevert("ReadImplementationAddresses: OPCM address has no code");
        script.run(input);
    }
}

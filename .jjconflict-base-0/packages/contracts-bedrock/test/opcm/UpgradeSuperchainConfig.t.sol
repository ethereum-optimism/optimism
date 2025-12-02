// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { UpgradeSuperchainConfig } from "scripts/deploy/UpgradeSuperchainConfig.s.sol";

// Interfaces
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";

import { DevFeatures } from "src/libraries/DevFeatures.sol";

/// @title MockOPCMV1
/// @notice This contract is used to mock the OPCM contract and emit an event which we check for in the test.
contract MockOPCMV1 {
    event UpgradeCalled(address indexed superchainConfig);

    function isDevFeatureEnabled(bytes32 /* _feature */ ) public pure returns (bool) {
        return false;
    }

    function upgradeSuperchainConfig(ISuperchainConfig _superchainConfig) public {
        emit UpgradeCalled(address(_superchainConfig));
    }
}

/// @title MockOPCMV2
/// @notice This contract is used to mock the OPCM v2 contract and emit an event which we check for in the test.
contract MockOPCMV2 {
    event UpgradeCalled(IOPContractsManagerV2.SuperchainUpgradeInput indexed superchainUpgradeInput);

    function isDevFeatureEnabled(bytes32 _feature) public pure returns (bool) {
        return _feature == DevFeatures.OPCM_V2;
    }

    function upgradeSuperchain(IOPContractsManagerV2.SuperchainUpgradeInput memory _superchainUpgradeInput) public {
        emit UpgradeCalled(_superchainUpgradeInput);
    }
}

/// @title UpgradeSuperchainConfig_Test
/// @notice This test is used to test the UpgradeSuperchainConfig script.
contract UpgradeSuperchainConfigV1_Run_Test is Test {
    MockOPCMV1 mockOPCM;
    UpgradeSuperchainConfig.Input input;
    UpgradeSuperchainConfig upgradeSuperchainConfig;
    address prank;
    ISuperchainConfig superchainConfig;

    event UpgradeCalled(address indexed superchainConfig);

    /// @notice Sets up the test suite.
    function setUp() public virtual {
        mockOPCM = new MockOPCMV1();

        input.opcm = address(mockOPCM);

        superchainConfig = ISuperchainConfig(makeAddr("superchainConfig"));
        prank = makeAddr("prank");

        input.superchainConfig = superchainConfig;
        input.prank = prank;

        upgradeSuperchainConfig = new UpgradeSuperchainConfig();
    }

    /// @notice Tests that the UpgradeSuperchainConfig script succeeds when called with non-zero input values.
    function test_upgrade_succeeds() public {
        // UpgradeCalled should be emitted by the prank since it's a delegate call.
        vm.expectEmit(address(prank));
        emit UpgradeCalled(address(superchainConfig));
        upgradeSuperchainConfig.run(input);
    }

    /// @notice Tests that the UpgradeSuperchainConfig script reverts when called with zero input values.
    function test_run_nullInput_reverts() public {
        input.prank = address(0);
        vm.expectRevert("UpgradeSuperchainConfig: prank not set");
        upgradeSuperchainConfig.run(input);
        input.prank = prank;

        input.opcm = address(0);
        vm.expectRevert("UpgradeSuperchainConfig: opcm not set");
        upgradeSuperchainConfig.run(input);
        input.opcm = address(mockOPCM);

        input.superchainConfig = ISuperchainConfig(address(0));
        vm.expectRevert("UpgradeSuperchainConfig: superchainConfig not set");
        upgradeSuperchainConfig.run(input);
        input.superchainConfig = ISuperchainConfig(address(superchainConfig));
    }
}

/// @title UpgradeSuperchainConfigV2_Run_Test
/// @notice This test is used to test the UpgradeSuperchainConfig script with OPCM v2.
contract UpgradeSuperchainConfigV2_Run_Test is Test {
    MockOPCMV2 mockOPCM;
    UpgradeSuperchainConfig upgradeSuperchainConfig;
    address prank;
    ISuperchainConfig superchainConfig;

    event UpgradeCalled(IOPContractsManagerV2.SuperchainUpgradeInput indexed superchainUpgradeInput);

    /// @notice Sets up the test suite.
    function setUp() public {
        mockOPCM = new MockOPCMV2();

        superchainConfig = ISuperchainConfig(makeAddr("superchainConfig"));
        prank = makeAddr("prank");

        upgradeSuperchainConfig = new UpgradeSuperchainConfig();
    }

    /// @notice Tests that the UpgradeSuperchainConfig script succeeds when called with non-zero input values.
    function testFuzz_upgrade_succeeds(IOPContractsManagerUtils.ExtraInstruction[] memory extraInstructions) public {
        UpgradeSuperchainConfig.Input memory input = _getInput(extraInstructions);

        // UpgradeCalled should be emitted by the prank since it's a delegate call.
        vm.expectEmit(address(prank));
        emit UpgradeCalled(
            IOPContractsManagerV2.SuperchainUpgradeInput({
                superchainConfig: superchainConfig,
                extraInstructions: extraInstructions
            })
        );
        upgradeSuperchainConfig.run(input);
    }

    function _getInput(IOPContractsManagerUtils.ExtraInstruction[] memory extraInstructions)
        internal
        view
        returns (UpgradeSuperchainConfig.Input memory)
    {
        return UpgradeSuperchainConfig.Input({
            prank: prank,
            opcm: address(mockOPCM),
            superchainConfig: superchainConfig,
            extraInstructions: extraInstructions
        });
    }
}

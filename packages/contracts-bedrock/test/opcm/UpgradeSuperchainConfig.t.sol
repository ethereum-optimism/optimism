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

/// @title MockOPCMV2
/// @notice This contract is used to mock the OPCM v2 contract and emit an event which we check for in the test.
contract MockOPCMV2 {
    event UpgradeCalled(IOPContractsManagerV2.SuperchainUpgradeInput indexed superchainUpgradeInput);

    function version() public pure returns (string memory) {
        return "7.0.0";
    }

    function upgradeSuperchain(IOPContractsManagerV2.SuperchainUpgradeInput memory _superchainUpgradeInput) public {
        emit UpgradeCalled(_superchainUpgradeInput);
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

    /// @notice Tests that the UpgradeSuperchainConfig script reverts when the OPCM v2 upgradeSuperchain
    /// call fails
    function test_upgrade_whenOPCMV2Reverts_reverts() public {
        UpgradeSuperchainConfig.Input memory input = _getInput(new IOPContractsManagerUtils.ExtraInstruction[](0));

        vm.mockCallRevert(
            prank,
            IOPContractsManagerV2.upgradeSuperchain.selector,
            abi.encode("UpgradeSuperchainConfig: upgrade failed")
        );

        vm.expectRevert("UpgradeSuperchainConfig: upgrade failed");
        upgradeSuperchainConfig.run(input);
    }
}

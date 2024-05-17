// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract dependencies
import "src/libraries/PortalErrors.sol";
import { OptimismPortalInterop } from "src/L1/OptimismPortalInterop.sol";
import { L1BlockInterop, ConfigType } from "src/L2/L1BlockInterop.sol";

contract OptimismPortalInterop_Test is CommonTest {
    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();
    }

    /// @dev Tests that the config for the gas paying token can be set.
    function testFuzz_setConfig_gasPayingToken_succeeds(bytes calldata _value) public {
        vm.expectEmit(address(optimismPortal));
        emitTransactionDeposited({
            _from: Constants.DEPOSITOR_ACCOUNT,
            _to: Predeploys.L1_BLOCK_ATTRIBUTES,
            _value: 0,
            _mint: 0,
            _gasLimit: 200_000,
            _isCreation: false,
            _data: abi.encodeCall(L1BlockInterop.setConfig, (ConfigType.GAS_PAYING_TOKEN, _value))
        });

        vm.prank(address(optimismPortal.systemConfig()));
        optimismPortal.setConfig(ConfigType.GAS_PAYING_TOKEN, _value);
    }

    /// @dev Tests that setting the gas paying token config as not the system config reverts.
    function testFuzz_setConfig_gasPayingToken_notSystemConfig_reverts(bytes calldata _value) public {
        vm.expectRevert(Unauthorized.selector);
        optimismPortal.setConfig(ConfigType.GAS_PAYING_TOKEN, _value);
    }

    /// @dev Tests that the config for adding a dependency can be set.
    function testFuzz_setConfig_addDependency_succeeds(bytes calldata _value) public {
        vm.expectEmit(address(optimismPortal));
        emitTransactionDeposited({
            _from: Constants.DEPOSITOR_ACCOUNT,
            _to: Predeploys.L1_BLOCK_ATTRIBUTES,
            _value: 0,
            _mint: 0,
            _gasLimit: 200_000,
            _isCreation: false,
            _data: abi.encodeCall(L1BlockInterop.setConfig, (ConfigType.ADD_DEPENDENCY, _value))
        });

        vm.prank(address(optimismPortal.systemConfig()));
        optimismPortal.setConfig(ConfigType.ADD_DEPENDENCY, _value);
    }

    /// @dev Tests that setting the add dependency config as not the system config reverts.
    function testFuzz_setConfig_addDependency_notSystemConfig_reverts(bytes calldata _value) public {
        vm.expectRevert(Unauthorized.selector);
        optimismPortal.setConfig(ConfigType.ADD_DEPENDENCY, _value);
    }

    /// @dev Tests that the config for removing a dependency can be set.
    function testFuzz_setConfig_removeDependency_succeeds(bytes calldata _value) public {
        vm.expectEmit(address(optimismPortal));
        emitTransactionDeposited({
            _from: Constants.DEPOSITOR_ACCOUNT,
            _to: Predeploys.L1_BLOCK_ATTRIBUTES,
            _value: 0,
            _mint: 0,
            _gasLimit: 200_000,
            _isCreation: false,
            _data: abi.encodeCall(L1BlockInterop.setConfig, (ConfigType.REMOVE_DEPENDENCY, _value))
        });

        vm.prank(address(optimismPortal.systemConfig()));
        optimismPortal.setConfig(ConfigType.REMOVE_DEPENDENCY, _value);
    }

    /// @dev Tests that setting the remove dependency config as not the system config reverts.
    function testFuzz_setConfig_removeDependency_notSystemConfig_reverts(bytes calldata _value) public {
        vm.expectRevert(Unauthorized.selector);
        optimismPortal.setConfig(ConfigType.REMOVE_DEPENDENCY, _value);
    }
}

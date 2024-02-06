// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { stdError } from "forge-std/Test.sol";

import { CommonTest } from "test/setup/CommonTest.sol";
import { NextImpl } from "test/mocks/NextImpl.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Constants } from "src/libraries/Constants.sol";

// Target contract dependencies
import { Proxy } from "src/universal/Proxy.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";

import "src/libraries/DisputeTypes.sol";

contract OptimismPortal2_Test is CommonTest {
    function setUp() public override {
        super.enableFaultProofs();
        super.setUp();
    }

    /// @dev Tests that the constructor sets the correct values.
    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function test_constructor_succeeds() external virtual {
        OptimismPortal2 opImpl = OptimismPortal2(payable(deploy.mustGetAddress("OptimismPortal2")));
        assertEq(address(opImpl.disputeGameFactory()), address(0));
        assertEq(address(opImpl.SYSTEM_CONFIG()), address(0));
        assertEq(address(opImpl.systemConfig()), address(0));
        assertEq(address(opImpl.superchainConfig()), address(0));
        assertEq(opImpl.l2Sender(), Constants.DEFAULT_L2_SENDER);
        assertEq(opImpl.respectedGameType().raw(), deploy.cfg().respectedGameType());
    }

    /// @dev Tests that the initializer sets the correct values.
    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function test_initialize_succeeds() external virtual {
        address guardian = deploy.cfg().superchainConfigGuardian();
        assertEq(address(optimismPortal2.disputeGameFactory()), address(disputeGameFactory));
        assertEq(address(optimismPortal2.SYSTEM_CONFIG()), address(systemConfig));
        assertEq(address(optimismPortal2.systemConfig()), address(systemConfig));
        assertEq(optimismPortal2.GUARDIAN(), guardian);
        assertEq(optimismPortal2.guardian(), guardian);
        assertEq(address(optimismPortal2.superchainConfig()), address(superchainConfig));
        assertEq(optimismPortal2.l2Sender(), Constants.DEFAULT_L2_SENDER);
        assertEq(optimismPortal2.paused(), false);
        assertEq(optimismPortal2.respectedGameType().raw(), deploy.cfg().respectedGameType());
    }

    /// @dev Tests that `pause` successfully pauses
    ///      when called by the GUARDIAN.
    function test_pause_succeeds() external {
        address guardian = optimismPortal2.GUARDIAN();

        assertEq(optimismPortal2.paused(), false);

        vm.expectEmit(address(superchainConfig));
        emit Paused("identifier");

        vm.prank(guardian);
        superchainConfig.pause("identifier");

        assertEq(optimismPortal2.paused(), true);
    }

    /// @dev Tests that `pause` reverts when called by a non-GUARDIAN.
    function test_pause_onlyGuardian_reverts() external {
        assertEq(optimismPortal2.paused(), false);

        assertTrue(optimismPortal2.GUARDIAN() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can pause");
        vm.prank(alice);
        superchainConfig.pause("identifier");

        assertEq(optimismPortal2.paused(), false);
    }

    /// @dev Tests that `unpause` successfully unpauses
    ///      when called by the GUARDIAN.
    function test_unpause_succeeds() external {
        address guardian = optimismPortal2.GUARDIAN();

        vm.prank(guardian);
        superchainConfig.pause("identifier");
        assertEq(optimismPortal2.paused(), true);

        vm.expectEmit(address(superchainConfig));
        emit Unpaused();
        vm.prank(guardian);
        superchainConfig.unpause();

        assertEq(optimismPortal2.paused(), false);
    }

    /// @dev Tests that `unpause` reverts when called by a non-GUARDIAN.
    function test_unpause_onlyGuardian_reverts() external {
        address guardian = optimismPortal2.GUARDIAN();

        vm.prank(guardian);
        superchainConfig.pause("identifier");
        assertEq(optimismPortal2.paused(), true);

        assertTrue(optimismPortal2.GUARDIAN() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can unpause");
        vm.prank(alice);
        superchainConfig.unpause();

        assertEq(optimismPortal2.paused(), true);
    }

    /// @dev Tests that `receive` successdully deposits ETH.
    function testFuzz_receive_succeeds(uint256 _value) external {
        vm.expectEmit(address(optimismPortal2));
        emitTransactionDeposited({
            _from: alice,
            _to: alice,
            _value: _value,
            _mint: _value,
            _gasLimit: 100_000,
            _isCreation: false,
            _data: hex""
        });

        // give alice money and send as an eoa
        vm.deal(alice, _value);
        vm.prank(alice, alice);
        (bool s,) = address(optimismPortal2).call{ value: _value }(hex"");

        assertTrue(s);
        assertEq(address(optimismPortal2).balance, _value);
    }

    /// @dev Tests that `depositTransaction` reverts when the destination address is non-zero
    ///      for a contract creation deposit.
    function test_depositTransaction_contractCreation_reverts() external {
        // contract creation must have a target of address(0)
        vm.expectRevert("OptimismPortal: must send to address(0) when creating a contract");
        optimismPortal2.depositTransaction(address(1), 1, 0, true, hex"");
    }

    /// @dev Tests that `depositTransaction` reverts when the data is too large.
    ///      This places an upper bound on unsafe blocks sent over p2p.
    function test_depositTransaction_largeData_reverts() external {
        uint256 size = 120_001;
        uint64 gasLimit = optimismPortal.minimumGasLimit(uint64(size));
        vm.expectRevert("OptimismPortal: data too large");
        optimismPortal2.depositTransaction({
            _to: address(0),
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: new bytes(size)
        });
    }

    /// @dev Tests that `depositTransaction` reverts when the gas limit is too small.
    function test_depositTransaction_smallGasLimit_reverts() external {
        vm.expectRevert("OptimismPortal: gas limit too small");
        optimismPortal2.depositTransaction({ _to: address(1), _value: 0, _gasLimit: 0, _isCreation: false, _data: hex"" });
    }
}

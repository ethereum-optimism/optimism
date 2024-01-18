// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { stdError } from "forge-std/Test.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";

// Target contract dependencies
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { DeploymentSummary } from "../proofs/utils/DeploymentSummary.sol";

/// @dev Contract testing the deployment summary correctness
contract DeploymentSummary_test is DeploymentSummary, CommonTest {
    address depositor;

    /// @notice super.setUp is not called on purpose
    function setUp() public override {
        // Recreate Deployment Summary state changes
        DeploymentSummary deploymentSummary = new DeploymentSummary();
        deploymentSummary.recreateDeployment();

        // Set summary addresses
        optimismPortal = OptimismPortal(payable(OptimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(SuperchainConfigProxyAddress);
        l2OutputOracle = L2OutputOracle(L2OutputOracleProxyAddress);
        systemConfig = SystemConfig(SystemConfigProxyAddress);

        // Set up utilized addresses
        depositor = makeAddr("depositor");
        alice = makeAddr("alice");
        bob = makeAddr("bob");
        vm.deal(alice, 10000 ether);
        vm.deal(bob, 10000 ether);
    }

    function test_summary_pause_succeeds() external {
        address guardian = optimismPortal.GUARDIAN();

        assertEq(optimismPortal.paused(), false);

        vm.expectEmit(address(superchainConfig));
        emit Paused("identifier");

        vm.prank(guardian);
        superchainConfig.pause("identifier");

        assertEq(optimismPortal.paused(), true);
    }

    function test_summary_pause_onlyGuardian_reverts() external {
        assertEq(optimismPortal.paused(), false);

        assertTrue(optimismPortal.GUARDIAN() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can pause");
        vm.prank(alice);
        superchainConfig.pause("identifier");

        assertEq(optimismPortal.paused(), false);
    }

    function test_unpause_succeeds() external {
        address guardian = optimismPortal.GUARDIAN();

        vm.prank(guardian);
        superchainConfig.pause("identifier");
        assertEq(optimismPortal.paused(), true);

        vm.expectEmit(address(superchainConfig));
        emit Unpaused();
        vm.prank(guardian);
        superchainConfig.unpause();

        assertEq(optimismPortal.paused(), false);
    }

    /// @dev Tests that `unpause` reverts when called by a non-GUARDIAN.
    function test_unpause_onlyGuardian_reverts() external {
        address guardian = optimismPortal.GUARDIAN();

        vm.prank(guardian);
        superchainConfig.pause("identifier");
        assertEq(optimismPortal.paused(), true);

        assertTrue(optimismPortal.GUARDIAN() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can unpause");
        vm.prank(alice);
        superchainConfig.unpause();

        assertEq(optimismPortal.paused(), true);
    }

    /// @dev Tests that `receive` successdully deposits ETH.
    function testFuzz_receive_succeeds(uint256 _value) external {
        vm.expectEmit(address(optimismPortal));
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
        (bool s,) = address(optimismPortal).call{ value: _value }(hex"");

        assertTrue(s);
        assertEq(address(optimismPortal).balance, _value);
    }

    /// @dev Tests that `depositTransaction` reverts when the destination address is non-zero
    ///      for a contract creation deposit.
    function test_depositTransaction_contractCreation_reverts() external {
        // contract creation must have a target of address(0)
        vm.expectRevert("OptimismPortal: must send to address(0) when creating a contract");
        optimismPortal.depositTransaction(address(1), 1, 0, true, hex"");
    }

    /// @dev Tests that `depositTransaction` reverts when the data is too large.
    ///      This places an upper bound on unsafe blocks sent over p2p.
    function test_depositTransaction_largeData_reverts() external {
        uint256 size = 120_001;
        uint64 gasLimit = optimismPortal.minimumGasLimit(uint64(size));
        vm.expectRevert("OptimismPortal: data too large");
        optimismPortal.depositTransaction({
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
        optimismPortal.depositTransaction({ _to: address(1), _value: 0, _gasLimit: 0, _isCreation: false, _data: hex"" });
    }

    /// @dev Tests that `depositTransaction` succeeds for small,
    ///      but sufficient, gas limits.
    function testFuzz_depositTransaction_smallGasLimit_succeeds(bytes memory _data, bool _shouldFail) external {
        uint64 gasLimit = optimismPortal.minimumGasLimit(uint64(_data.length));
        if (_shouldFail) {
            gasLimit = uint64(bound(gasLimit, 0, gasLimit - 1));
            vm.expectRevert("OptimismPortal: gas limit too small");
        }

        optimismPortal.depositTransaction({
            _to: address(0x40),
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: _data
        });
    }

    /// @dev Tests that `minimumGasLimit` succeeds for small calldata sizes.
    ///      The gas limit should be 21k for 0 calldata and increase linearly
    ///      for larger calldata sizes.
    function test_minimumGasLimit_succeeds() external {
        assertEq(optimismPortal.minimumGasLimit(0), 21_000);
        assertTrue(optimismPortal.minimumGasLimit(2) > optimismPortal.minimumGasLimit(1));
        assertTrue(optimismPortal.minimumGasLimit(3) > optimismPortal.minimumGasLimit(2));
    }

    /// @dev Tests that `depositTransaction` succeeds for an EOA.
    function testFuzz_depositTransaction_eoa_succeeds(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        uint256 _mint,
        bool _isCreation,
        bytes memory _data
    )
        external
    {
        _gasLimit = uint64(
            bound(
                _gasLimit,
                optimismPortal.minimumGasLimit(uint64(_data.length)),
                systemConfig.resourceConfig().maxResourceLimit
            )
        );
        if (_isCreation) _to = address(0);

        // EOA emulation
        vm.expectEmit(address(optimismPortal));
        emitTransactionDeposited({
            _from: depositor,
            _to: _to,
            _value: _value,
            _mint: _mint,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });

        vm.deal(depositor, _mint);
        vm.prank(depositor, depositor);
        optimismPortal.depositTransaction{ value: _mint }({
            _to: _to,
            _value: _value,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });
        assertEq(address(optimismPortal).balance, _mint);
    }

    /// @dev Tests that `depositTransaction` succeeds for a contract.
    function testFuzz_depositTransaction_contract_succeeds(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        uint256 _mint,
        bool _isCreation,
        bytes memory _data
    )
        external
    {
        _gasLimit = uint64(
            bound(
                _gasLimit,
                optimismPortal.minimumGasLimit(uint64(_data.length)),
                systemConfig.resourceConfig().maxResourceLimit
            )
        );
        if (_isCreation) _to = address(0);

        vm.expectEmit(address(optimismPortal));
        emitTransactionDeposited({
            _from: AddressAliasHelper.applyL1ToL2Alias(address(this)),
            _to: _to,
            _value: _value,
            _mint: _mint,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });

        vm.deal(address(this), _mint);
        vm.prank(address(this));
        optimismPortal.depositTransaction{ value: _mint }({
            _to: _to,
            _value: _value,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });
        assertEq(address(optimismPortal).balance, _mint);
    }

    /// @dev Tests that `isOutputFinalized` succeeds for an EOA depositing a tx with ETH and data.
    function testFail_simple_isOutputFinalized_succeeds() external {
        uint256 startingBlockNumber = deploy.cfg().l2OutputOracleStartingBlockNumber();

        uint256 ts = block.timestamp;
        vm.mockCall(
            address(optimismPortal.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(Types.OutputProposal(bytes32(uint256(1)), uint128(ts), uint128(startingBlockNumber)))
        );

        // warp to the finalization period
        vm.warp(ts + l2OutputOracle.FINALIZATION_PERIOD_SECONDS());
        assertEq(optimismPortal.isOutputFinalized(0), false);

        // warp past the finalization period
        vm.warp(ts + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);
        assertEq(optimismPortal.isOutputFinalized(0), true);
    }

    /// @dev Tests `isOutputFinalized` for a finalized output.
    function testFail_isOutputFinalized_succeeds() external {
        uint256 checkpoint = l2OutputOracle.nextBlockNumber();
        uint256 nextOutputIndex = l2OutputOracle.nextOutputIndex();
        vm.roll(checkpoint);
        vm.warp(l2OutputOracle.computeL2Timestamp(checkpoint) + 1);
        vm.prank(l2OutputOracle.PROPOSER());
        l2OutputOracle.proposeL2Output(keccak256(abi.encode(2)), checkpoint, 0, 0);

        // warp to the final second of the finalization period
        uint256 finalizationHorizon = block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS();
        vm.warp(finalizationHorizon);
        // The checkpointed block should not be finalized until 1 second from now.
        assertEq(optimismPortal.isOutputFinalized(nextOutputIndex), false);
        // Nor should a block after it
        vm.expectRevert(stdError.indexOOBError);
        assertEq(optimismPortal.isOutputFinalized(nextOutputIndex + 1), false);
        // warp past the finalization period
        vm.warp(finalizationHorizon + 1);
        // It should now be finalized.
        assertEq(optimismPortal.isOutputFinalized(nextOutputIndex), true);
        // But not the block after it.
        vm.expectRevert(stdError.indexOOBError);
        assertEq(optimismPortal.isOutputFinalized(nextOutputIndex + 1), false);
    }
}

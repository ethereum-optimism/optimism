//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { AddressAliasHelper } from "@eth-optimism/contracts/standards/AddressAliasHelper.sol";

import { CommonTest } from "./CommonTest.t.sol";

import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { WithdrawalVerifier } from "../libraries/Lib_WithdrawalVerifier.sol";

contract OptimismPortal_Test is CommonTest {
    event TransactionDeposited(
        address indexed from,
        address indexed to,
        uint256 mint,
        uint256 value,
        uint64 gasLimit,
        bool isCreation,
        bytes data
    );

    // Dependencies
    L2OutputOracle oracle;
    OptimismPortal op;

    function setUp() external {
        oracle = new L2OutputOracle(
            1800,
            2,
            keccak256(abi.encode(0)),
            100,
            1,
            address(666)
        );
        op = new OptimismPortal(oracle, 7 days);
    }

    function test_OptimismPortalConstructor() external {
        assertEq(op.FINALIZATION_PERIOD(), 7 days);
        assertEq(address(op.L2_ORACLE()), address(oracle));
        assertEq(op.l2Sender(), 0x000000000000000000000000000000000000dEaD);
    }

    function test_OptimismPortalReceiveEth() external {
        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(
            alice,
            alice,
            100,
            100,
            100_000,
            false,
            hex""
        );

        // give alice money and send as an eoa
        vm.deal(alice, 2**64);
        vm.prank(alice, alice);
        (bool s, ) = address(op).call{ value: 100 }(hex"");

        assert(s);
        assertEq(address(op).balance, 100);
    }

    // function test_OptimismPortalDepositTransaction() external {}

    // Test: depositTransaction fails when contract creation has a non-zero destination address
    function test_OptimismPortalContractCreationReverts() external {
        // contract creation must have a target of address(0)
        vm.expectRevert(abi.encodeWithSignature("NonZeroCreationTarget()"));
        op.depositTransaction(address(1), 1, 0, true, hex"");
    }

    // Test: depositTransaction should emit the correct log when an EOA deposits a tx with 0 value
    function test_depositTransaction_NoValueEOA() external {
        // EOA emulation
        vm.prank(address(this), address(this));
        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(
            address(this),
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );

        op.depositTransaction(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }

    // Test: depositTransaction should emit the correct log when a contract deposits a tx with 0 value
    function test_depositTransaction_NoValueContract() external {
        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(
            AddressAliasHelper.applyL1ToL2Alias(address(this)),
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );

        op.depositTransaction(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }

    // Test: depositTransaction should emit the correct log when an EOA deposits a contract creation with 0 value
    function test_depositTransaction_createWithZeroValueForEOA() external {
        // EOA emulation
        vm.prank(address(this), address(this));

        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(
            address(this),
            ZERO_ADDRESS,
            ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            NON_ZERO_DATA
        );

        op.depositTransaction(ZERO_ADDRESS, ZERO_VALUE, NON_ZERO_GASLIMIT, true, NON_ZERO_DATA);
    }

    // Test: depositTransaction should emit the correct log when a contract deposits a contract creation with 0 value
    function test_depositTransaction_createWithZeroValueForContract() external {
        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(
            AddressAliasHelper.applyL1ToL2Alias(address(this)),
            ZERO_ADDRESS,
            ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            NON_ZERO_DATA
        );

        op.depositTransaction(ZERO_ADDRESS, ZERO_VALUE, NON_ZERO_GASLIMIT, true, NON_ZERO_DATA);
    }

    // Test: depositTransaction should increase its eth balance when an EOA deposits a transaction with ETH
    function test_depositTransaction_withEthValueFromEOA() external {
        // EOA emulation
        vm.prank(address(this), address(this));

        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(
            address(this),
            NON_ZERO_ADDRESS,
            NON_ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );

        op.depositTransaction{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
        assertEq(address(op).balance, NON_ZERO_VALUE);
    }

    // Test: depositTransaction should increase its eth balance when a contract deposits a transaction with ETH
    function test_depositTransaction_withEthValueFromContract() external {
        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(
            AddressAliasHelper.applyL1ToL2Alias(address(this)),
            NON_ZERO_ADDRESS,
            NON_ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );

        op.depositTransaction{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }

    // Test: depositTransaction should increase its eth balance when an EOA deposits a contract creation with ETH
    function test_depositTransaction_withEthValueAndEOAContractCreation() external {
        // EOA emulation
        vm.prank(address(this), address(this));

        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(
            address(this),
            ZERO_ADDRESS,
            NON_ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            hex""
        );

        op.depositTransaction{ value: NON_ZERO_VALUE }(
            ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            hex""
        );
        assertEq(address(op).balance, NON_ZERO_VALUE);
    }

    // Test: depositTransaction should increase its eth balance when a contract deposits a contract creation with ETH
    function test_depositTransaction_withEthValueAndContractContractCreation() external {
        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(
            AddressAliasHelper.applyL1ToL2Alias(address(this)),
            ZERO_ADDRESS,
            NON_ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            NON_ZERO_DATA
        );

        op.depositTransaction{ value: NON_ZERO_VALUE }(
            ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            NON_ZERO_DATA
        );
        assertEq(address(op).balance, NON_ZERO_VALUE);
    }

    // TODO: test this deeply
    // function test_verifyWithdrawal() external {}

    function test_cannotVerifyRecentWithdrawal() external {
        WithdrawalVerifier.OutputRootProof memory outputRootProof = WithdrawalVerifier.OutputRootProof({
            version: bytes32(0),
            stateRoot: bytes32(0),
            withdrawerStorageRoot: bytes32(0),
            latestBlockhash: bytes32(0)
        });

        vm.expectRevert("Proposal is not yet finalized.");
        op.finalizeWithdrawalTransaction(
            0,
            alice,
            alice,
            0,
            0,
            hex"",
            0,
            outputRootProof,
            hex""
        );
    }

    function test_invalidWithdrawalProof() external {
        WithdrawalVerifier.OutputRootProof memory outputRootProof = WithdrawalVerifier.OutputRootProof({
            version: bytes32(0),
            stateRoot: bytes32(0),
            withdrawerStorageRoot: bytes32(0),
            latestBlockhash: bytes32(0)
        });

        vm.warp(oracle.nextTimestamp() + op.FINALIZATION_PERIOD());
        vm.expectRevert(abi.encodeWithSignature("InvalidOutputRootProof()"));
        op.finalizeWithdrawalTransaction(
            0,
            alice,
            alice,
            0,
            0,
            hex"",
            0,
            outputRootProof,
            hex""
        );
    }
}

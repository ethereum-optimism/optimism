//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Portal_Initializer, CommonTest, NextImpl } from "./CommonTest.t.sol";

import { AddressAliasHelper } from "../libraries/AddressAliasHelper.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { WithdrawalVerifier } from "../libraries/Lib_WithdrawalVerifier.sol";
import { Proxy } from "../universal/Proxy.sol";

contract OptimismPortal_Test is Portal_Initializer {
    event TransactionDeposited(
        address indexed from,
        address indexed to,
        uint256 mint,
        uint256 value,
        uint64 gasLimit,
        bool isCreation,
        bytes data
    );

    function test_OptimismPortalConstructor() external {
        assertEq(op.FINALIZATION_PERIOD_SECONDS(), 7 days);
        assertEq(address(op.L2_ORACLE()), address(oracle));
        assertEq(op.l2Sender(), 0x000000000000000000000000000000000000dEaD);
    }

    function test_OptimismPortalReceiveEth() external {
        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(alice, alice, 100, 100, 100_000, false, hex"");

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
        vm.expectRevert("OptimismPortal: must send to address(0) when creating a contract");
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
        WithdrawalVerifier.OutputRootProof memory outputRootProof = WithdrawalVerifier
            .OutputRootProof({
                version: bytes32(0),
                stateRoot: bytes32(0),
                withdrawerStorageRoot: bytes32(0),
                latestBlockhash: bytes32(0)
            });

        vm.expectRevert("OptimismPortal: proposal is not yet finalized");
        op.finalizeWithdrawalTransaction(0, alice, alice, 0, 0, hex"", 0, outputRootProof, hex"");
    }

    function test_invalidWithdrawalProof() external {
        WithdrawalVerifier.OutputRootProof memory outputRootProof = WithdrawalVerifier
            .OutputRootProof({
                version: bytes32(0),
                stateRoot: bytes32(0),
                withdrawerStorageRoot: bytes32(0),
                latestBlockhash: bytes32(0)
            });

        vm.warp(
            oracle.getL2Output(
                oracle.latestBlockNumber()
            ).timestamp
            + op.FINALIZATION_PERIOD_SECONDS()
        );

        vm.expectRevert("OptimismPortal: invalid output root proof");
        op.finalizeWithdrawalTransaction(0, alice, alice, 0, 0, hex"", 0, outputRootProof, hex"");
    }

    function test_simple_isOutputFinalized() external {
        vm.mockCall(
            address(op.L2_ORACLE()),
            abi.encodeWithSelector(
                L2OutputOracle.getL2Output.selector
            ),
            abi.encode(
                L2OutputOracle.OutputProposal(
                    bytes32(uint256(1)),
                    0
                )
            )
        );

        // warp to the finalization period
        vm.warp(op.FINALIZATION_PERIOD_SECONDS());
        assertEq(op.isOutputFinalized(0), false);
        // warp past the finalization period
        vm.warp(op.FINALIZATION_PERIOD_SECONDS() + 1);
        assertEq(op.isOutputFinalized(0), true);
    }

    function test_isOutputFinalized() external {
        uint256 checkpoint = oracle.nextBlockNumber();
        vm.roll(checkpoint);
        vm.warp(oracle.computeL2Timestamp(checkpoint) + 1);
        vm.prank(oracle.sequencer());
        oracle.appendL2Output(keccak256(abi.encode(2)), checkpoint, 0, 0);

        // warp to the final second of the finalization period
        uint256 finalizationHorizon = block.timestamp + op.FINALIZATION_PERIOD_SECONDS();
        vm.warp(finalizationHorizon);
        // The checkpointed block should not be finalized until 1 second from now.
        assertEq(op.isOutputFinalized(checkpoint), false);
        // Nor should a block after it
        assertEq(op.isOutputFinalized(checkpoint + 1), false);
        // Nor a block before it, even though the finalization period has passed, there is
        // not yet a checkpoint block on top of it for which that is true.
        assertEq(op.isOutputFinalized(checkpoint - 1), false);

        // warp past the finalization period
        vm.warp(finalizationHorizon + 1);
        // It should now be finalized.
        assertEq(op.isOutputFinalized(checkpoint), true);
        // So should the block before it.
        assertEq(op.isOutputFinalized(checkpoint - 1), true);
        // But not the block after it.
        assertEq(op.isOutputFinalized(checkpoint + 1), false);
    }
}

contract OptimismPortalUpgradeable_Test is Portal_Initializer {
    Proxy internal proxy;
    uint64 initialBlockNum;

    function setUp() public override {
        super.setUp();
        initialBlockNum = uint64(block.number);
        proxy = Proxy(payable(address(op)));
    }

    function test_initValuesOnProxy() external {
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = OptimismPortal(
            payable(address(proxy))
        ).params();
        assertEq(prevBaseFee, opImpl.INITIAL_BASE_FEE());
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum);
    }

    function test_cannotInitProxy() external {
        vm.expectRevert("Initializable: contract is already initialized");
        address(proxy).call(abi.encodeWithSelector(OptimismPortal.initialize.selector));
    }

    function test_cannotInitImpl() external {
        vm.expectRevert("Initializable: contract is already initialized");
        address(opImpl).call(abi.encodeWithSelector(OptimismPortal.initialize.selector));
    }

    function test_upgrading() external {
        // Check an unused slot before upgrading.
        bytes32 slot21Before = vm.load(address(op), bytes32(uint256(21)));
        assertEq(bytes32(0), slot21Before);

        NextImpl nextImpl = new NextImpl();
        vm.startPrank(alice);
        proxy.upgradeToAndCall(
            address(nextImpl),
            abi.encodeWithSelector(NextImpl.initialize.selector)
        );
        assertEq(proxy.implementation(), address(nextImpl));

        // Verify that the NextImpl contract initialized its values according as expected
        bytes32 slot21After = vm.load(address(op), bytes32(uint256(21)));
        bytes32 slot21Expected = NextImpl(address(op)).slot21Init();
        assertEq(slot21Expected, slot21After);
    }
}

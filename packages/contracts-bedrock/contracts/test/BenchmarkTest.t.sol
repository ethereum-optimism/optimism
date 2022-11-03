// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import "./CommonTest.t.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { ResourceMetering } from "../L1/ResourceMetering.sol";

uint128 constant INITIAL_BASE_FEE = 1_000_000_000;

// Free function for setting the prevBaseFee param in the OptimismPortal.
function setPrevBaseFee(
    Vm _vm,
    address _op,
    uint128 _prevBaseFee
) {
    _vm.store(
        address(_op),
        bytes32(uint256(1)),
        bytes32(
            abi.encode(
                ResourceMetering.ResourceParams({
                    prevBaseFee: _prevBaseFee,
                    prevBoughtGas: 0,
                    prevBlockNum: uint64(block.number)
                })
            )
        )
    );
}

// Tests for obtaining pure gas cost estimates for commonly used functions.
// The objective with these benchmarks is to strip down the actual test functions
// so that they are nothing more than the call we want measure the gas cost of.
// In order to achieve this we make no assertions, and handle everything else in the setUp()
// function.
contract GasBenchMark_OptimismPortal is Portal_Initializer {
    function test_depositTransaction_benchmark() external {
        op.depositTransaction{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }

    function test_depositTransaction_benchmark_1() external {
        setPrevBaseFee(vm, address(op), INITIAL_BASE_FEE);
        op.depositTransaction{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }
}

contract GasBenchMark_L1CrossDomainMessenger is Messenger_Initializer {
    function test_L1MessengerSendMessage_benchmark_0() external {
        // The amount of data typically sent during a bridge deposit.
        bytes
            memory data = hex"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff";
        L1Messenger.sendMessage(bob, data, uint32(100));
    }

    function test_L1MessengerSendMessage_benchmark_1() external {
        setPrevBaseFee(vm, address(op), INITIAL_BASE_FEE);
        // The amount of data typically sent during a bridge deposit.
        bytes
            memory data = hex"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff";
        L1Messenger.sendMessage(bob, data, uint32(100));
    }
}

contract GasBenchMark_L1StandardBridge_Deposit is Bridge_Initializer {
    function setUp() public virtual override {
        super.setUp();
        deal(address(L1Token), alice, 100000, true);
        vm.startPrank(alice, alice);
    }

    function test_depositETH_benchmark_0() external {
        L1Bridge.depositETH{ value: 500 }(50000, hex"");
    }

    function test_depositETH_benchmark_1() external {
        setPrevBaseFee(vm, address(op), INITIAL_BASE_FEE);
        L1Bridge.depositETH{ value: 500 }(50000, hex"");
    }

    function test_depositERC20_benchmark_0() external {
        L1Bridge.depositETH{ value: 500 }(50000, hex"");
    }

    function test_depositERC20_benchmark_1() external {
        setPrevBaseFee(vm, address(op), INITIAL_BASE_FEE);
        L1Bridge.depositETH{ value: 500 }(50000, hex"");
    }
}

contract GasBenchMark_L1StandardBridge_Finalize is Bridge_Initializer {
    function setUp() public virtual override {
        super.setUp();
        deal(address(L1Token), address(L1Bridge), 100, true);
        vm.mockCall(
            address(L1Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.OTHER_BRIDGE()))
        );
        vm.startPrank(address(L1Bridge.messenger()));
        vm.deal(address(L1Bridge.messenger()), 100);
    }

    function test_finalizeETHWithdrawal_benchmark() external {
        // TODO: Make this more accurate. It is underestimating the cost because it pranks
        // the call coming from the messenger, which bypasses the portal
        // and oracle.
        L1Bridge.finalizeETHWithdrawal{ value: 100 }(alice, alice, 100, hex"");
    }
}

contract GasBenchMark_L2OutputOracle is L2OutputOracle_Initializer {
    uint256 nextBlockNumber;

    function setUp() public override {
        super.setUp();
        nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.startPrank(proposer);
    }

    function test_proposeL2Output_benchmark() external {
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber, 0, 0);
    }
}

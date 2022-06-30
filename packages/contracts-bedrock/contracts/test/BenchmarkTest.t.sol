//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/* Testing utilities */
import "./CommonTest.t.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";

// Tests for obtaining pure gas cost estimates for commonly used functions.
// The objective with these benchmarks is to strip down the actual test functions
// so that they are nothing more than the call we want measure the gas cost of.
// In order to achieve this we make no assertions, and handle everything else in the setUp()
// function.
contract GasBenchMark_L1CrossDomainMessenger is Messenger_Initializer {
    function test_L1MessengerSendMessage_benchmark() external {
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

    function test_depositETH_benchmark() external {
        L1Bridge.depositETH{ value: 500 }(50000, hex"");
    }

    function test_depositERC20_benchmark() external {
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
            abi.encode(address(L1Bridge.otherBridge()))
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

contract GasBenchMark_OptimismPortal is Portal_Initializer {
    function setUp() public override {
        super.setUp();
    }

    function test_depositTransaction_benchmark() external {
        op.depositTransaction{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }
}

contract GasBenchMark_L2OutputOracle is L2OutputOracle_Initializer {
    uint256 nextBlockNumber;

    function setUp() public override {
        super.setUp();
        nextBlockNumber = oracle.nextBlockNumber();
        warpToAppendTime(nextBlockNumber);
        vm.startPrank(sequencer);
    }

    function test_appendL2Output_benchmark() external {
        oracle.appendL2Output(nonZeroHash, nextBlockNumber, 0, 0);
    }
}

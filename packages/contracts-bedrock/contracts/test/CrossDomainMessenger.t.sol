// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Messenger_Initializer, Reverter, CallerCaller, CommonTest } from "./CommonTest.t.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";

// Libraries
import { Predeploys } from "../libraries/Predeploys.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Encoding } from "../libraries/Encoding.sol";

// CrossDomainMessenger_Test is for testing functionality which is common to both the L1 and L2
// CrossDomainMessenger contracts. For simplicity, we use the L1 Messenger as the test contract.
contract CrossDomainMessenger_BaseGas_Test is Messenger_Initializer {
    // Ensure that baseGas passes for the max value of _minGasLimit,
    // this is about 4 Billion.
    function test_baseGas_succeeds() external view {
        L1Messenger.baseGas(hex"ff", type(uint32).max);
    }

    // Fuzz for other values which might cause a revert in baseGas.
    function testFuzz_baseGas_succeeds(uint32 _minGasLimit) external view {
        L1Messenger.baseGas(hex"ff", _minGasLimit);
    }
}

/**
 * @title ExternalRelay
 * @notice A mock external contract called via the SafeCall inside
 *         the CrossDomainMessenger's `relayMessage` function.
 */
contract ExternalRelay is CommonTest {
    address internal op;
    address internal fuzzedSender;
    L1CrossDomainMessenger internal L1Messenger;

    event FailedRelayedMessage(bytes32 indexed msgHash);

    constructor(L1CrossDomainMessenger _l1Messenger, address _op) {
        L1Messenger = _l1Messenger;
        op = _op;
    }

    /**
     * @notice Internal helper function to relay a message and perform assertions.
     */
    function _internalRelay(address _innerSender) internal {
        address initialSender = L1Messenger.xDomainMessageSender();

        bytes memory callMessage = getCallData();

        bytes32 hash = Hashing.hashCrossDomainMessage({
            _nonce: Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            _sender: _innerSender,
            _target: address(this),
            _value: 0,
            _gasLimit: 0,
            _data: callMessage
        });

        vm.expectEmit(true, true, true, true);
        emit FailedRelayedMessage(hash);

        vm.prank(address(op));
        L1Messenger.relayMessage({
            _nonce: Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            _sender: _innerSender,
            _target: address(this),
            _value: 0,
            _minGasLimit: 0,
            _message: callMessage
        });

        assertTrue(L1Messenger.failedMessages(hash));
        assertFalse(L1Messenger.successfulMessages(hash));
        assertEq(initialSender, L1Messenger.xDomainMessageSender());
    }

    /**
     * @notice externalCallWithMinGas is called by the CrossDomainMessenger.
     */
    function externalCallWithMinGas() external payable {
        for (uint256 i = 0; i < 10; i++) {
            address _innerSender;
            unchecked {
                _innerSender = address(uint160(uint256(uint160(fuzzedSender)) + i));
            }
            _internalRelay(_innerSender);
        }
    }

    /**
     * @notice Helper function to get the callData for an `externalCallWithMinGas
     */
    function getCallData() public pure returns (bytes memory) {
        return abi.encodeWithSelector(ExternalRelay.externalCallWithMinGas.selector);
    }

    /**
     * @notice Helper function to set the fuzzed sender
     */
    function setFuzzedSender(address _fuzzedSender) public {
        fuzzedSender = _fuzzedSender;
    }
}

/**
 * @title CrossDomainMessenger_RelayMessage_Test
 * @notice Fuzz tests re-entrancy into the CrossDomainMessenger relayMessage function.
 */
contract CrossDomainMessenger_RelayMessage_Test is Messenger_Initializer {
    // Storage slot of the l2Sender
    uint256 constant senderSlotIndex = 50;

    ExternalRelay public er;

    function setUp() public override {
        super.setUp();
        er = new ExternalRelay(L1Messenger, address(op));
    }

    /**
     * @dev This test mocks an OptimismPortal call to the L1CrossDomainMessenger via
     *      the relayMessage function. The relayMessage function will then use SafeCall's
     *      callWithMinGas to call the target with call data packed in the callMessage.
     *      For this test, the callWithMinGas will call the mock ExternalRelay test contract
     *      defined above, executing the externalCallWithMinGas function which will try to
     *      re-enter the CrossDomainMessenger's relayMessage function, resulting in that message
     *      being recorded as failed.
     */
    function testFuzz_relayMessageReenter_succeeds(address _sender, uint256 _gasLimit) external {
        vm.assume(_sender != Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        er.setFuzzedSender(_sender);
        address target = address(er);
        bytes memory callMessage = er.getCallData();

        vm.expectCall(target, callMessage);

        uint64 gasLimit = uint64(bound(_gasLimit, 0, 30_000_000));

        bytes32 hash = Hashing.hashCrossDomainMessage({
            _nonce: Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            _sender: sender,
            _target: target,
            _value: 0,
            _gasLimit: gasLimit,
            _data: callMessage
        });

        // set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));
        vm.prank(address(op));
        L1Messenger.relayMessage({
            _nonce: Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            _sender: sender,
            _target: target,
            _value: 0,
            _minGasLimit: gasLimit,
            _message: callMessage
        });

        assertTrue(L1Messenger.successfulMessages(hash));
        assertEq(L1Messenger.failedMessages(hash), false);

        // Ensures that the `xDomainMsgSender` is set back to `Predeploys.L2_CROSS_DOMAIN_MESSENGER`
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        L1Messenger.xDomainMessageSender();
    }
}

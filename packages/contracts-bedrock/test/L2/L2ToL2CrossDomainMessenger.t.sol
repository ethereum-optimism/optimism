// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Bridge_Initializer } from "test/setup/Bridge_Initializer.sol";
import { Reverter, ConfigurableCaller } from "test/mocks/Callers.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Types } from "src/libraries/Types.sol";
import { ICrossL2Inbox, IL2ToL2CrossDomainMessenger } from "src/libraries/Predeploys.sol";

// Target contract dependencies
import { L2ToL1MessagePasser } from "src/L2/L2ToL1MessagePasser.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

contract L2ToL2CrossDomainMessengerTest is Bridge_Initializer {
    /// @dev Target address for testing
    address target = address(0xabcd);

    /// @dev Tests that the implementation is initialized correctly.
    function test_constructor_succeeds() external {
        assertEq(l2ToL2CrossDomainMessenger.MESSAGE_VERSION(), uint16(0));
        assertEq(l2ToL2CrossDomainMessenger.messageNonce(), uint256(0));
    }

    /// @dev Tests that `messageNonce` can be decoded correctly.
    function test_messageVersion_succeeds() external {
        assertEq(address(l2ToL2CrossDomainMessenger), 0x4200000000000000000000000000000000000023);
        (, uint16 version) = Encoding.decodeVersionedNonce(l2ToL2CrossDomainMessenger.messageNonce());
        assertEq(version, l2ToL2CrossDomainMessenger.MESSAGE_VERSION());
    }

    /// @dev Tests that `sendMessage` executes successfully.
    function testFuzz_sendMessage_succeeds(uint256 _destination, address _target) external {
        vm.assume(_destination != block.chainid);

        bytes memory xDomainCallData = hex"aa";

        l2ToL2CrossDomainMessenger.sendMessage(_destination, _target, xDomainCallData);
    }

    /// @dev Tests that `sendMessage` can be called twice and that
    ///      the nonce increments correctly.
    function test_sendMessage_twice_succeeds() external {
        uint256 destination = 123 == block.chainid ? 456 : 123;
        bytes memory xDomainCallData = hex"aa";
        uint256 nonce = l2ToL2CrossDomainMessenger.messageNonce();
        l2ToL2CrossDomainMessenger.sendMessage(destination, target, xDomainCallData);
        l2ToL2CrossDomainMessenger.sendMessage(destination, target, xDomainCallData);
        // the nonce increments for each message sent
        assertEq(nonce + 2, l2ToL2CrossDomainMessenger.messageNonce());
    }

    /// @dev Tests that `relayMessage` executes successfully.
    function test_relayMessage_succeeds() external {
        address sender = address(l2ToL2CrossDomainMessenger);

        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: address(l2ToL2CrossDomainMessenger),
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: block.chainid
        });

        vm.prank(tx.origin);
        crossL2Inbox.executeMessage(id, target, hex"1111");

        vm.prank(address(crossL2Inbox));
        l2ToL2CrossDomainMessenger.relayMessage(
            block.chainid,
            block.chainid,
            Encoding.encodeVersionedNonce(0, 1), // nonce
            sender,
            target,
            hex"1111"
        );

        assert(
            l2ToL2CrossDomainMessenger.successfulMessages(
                keccak256(
                    abi.encode(
                        block.chainid, block.chainid, Encoding.encodeVersionedNonce(0, 1), sender, target, hex"1111"
                    )
                )
            )
        );
    }
}

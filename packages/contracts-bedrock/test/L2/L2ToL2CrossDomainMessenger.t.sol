// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Bridge_Initializer } from "test/setup/Bridge_Initializer.sol";
import { Reverter, ConfigurableCaller } from "test/mocks/Callers.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { ICrossL2Inbox } from "src/L2/ICrossL2Inbox.sol";

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Types } from "src/libraries/Types.sol";

// Target contract dependencies
import { L2ToL1MessagePasser } from "src/L2/L2ToL1MessagePasser.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

contract L2ToL2CrossDomainMessengerTest is Bridge_Initializer {
    /// @dev Target address for testing
    address target = address(0xabcd);

    /// @dev Tests that `sendMessage` executes successfully.
    function testFuzz_sendMessage_succeeds(uint256 _destination, address _target) external {
        vm.assume(_destination != block.chainid);

        bytes memory xDomainCallData = hex"aa";

        l2ToL2CrossDomainMessenger.sendMessage(_destination, _target, xDomainCallData);
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

        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), sender);

        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), block.chainid);
    }
}

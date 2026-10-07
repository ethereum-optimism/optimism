// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { IUndeliveredMessageExporter } from "interfaces/L2/IUndeliveredMessageExporter.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title UndeliveredMessageExporter_TestInit
/// @notice Reusable test initialization for `UndeliveredMessageExporter` tests.
abstract contract UndeliveredMessageExporter_TestInit is CommonTest {
    IUndeliveredMessageExporter internal exporter =
        IUndeliveredMessageExporter(Predeploys.UNDELIVERED_MESSAGE_EXPORTER);
    IL2ToL2CrossDomainMessenger internal messenger =
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

    /// @notice A message to another chain's L1CrossDomainMessenger, as the exporter sends it.
    address internal sourceMessenger = makeAddr("sourceMessenger");

    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();
    }
}

/// @title UndeliveredMessageExporter_ExportUndeliveredMessage_Test
/// @notice Tests the `exportUndeliveredMessage` function of the `UndeliveredMessageExporter` contract.
contract UndeliveredMessageExporter_ExportUndeliveredMessage_Test is UndeliveredMessageExporter_TestInit {
    /// @notice Selector of the L2ToL2CrossDomainMessenger's SentMessage event.
    bytes32 internal constant SENT_MESSAGE_EVENT_SELECTOR =
        keccak256("SentMessage(uint256,address,uint256,address,bytes)");

    /// @notice Tests that an undelivered message is sent, as this predeploy, through the
    ///         L2CrossDomainMessenger to the named source L1CrossDomainMessenger, hashed with this chain
    ///         as the destination.
    function testFuzz_exportUndeliveredMessage_succeeds(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
    {
        bytes32 messageHash =
            Hashing.hashL2toL2CrossDomainMessage(block.chainid, _source, _nonce, _sender, _target, _message);
        vm.expectEmit(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        emit SentMessage(
            sourceMessenger,
            Predeploys.UNDELIVERED_MESSAGE_EXPORTER,
            abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (messageHash, block.timestamp)),
            l2CrossDomainMessenger.messageNonce(),
            _minGasLimit
        );

        bytes32 returned = exporter.exportUndeliveredMessage(
            sourceMessenger, _source, _nonce, _sender, _target, _message, _minGasLimit
        );

        assertEq(returned, messageHash);
    }

    /// @notice Tests that the export works from a deposit, so no sequencer can keep it out.
    function test_exportUndeliveredMessage_fromDeposit_succeeds() external {
        address depositor = AddressAliasHelper.applyL1ToL2Alias(alice);
        uint256 nonce = l2CrossDomainMessenger.messageNonce();

        vm.prank(depositor, depositor);
        exporter.exportUndeliveredMessage(sourceMessenger, 1, 0, alice, bob, hex"1234", 1_000_000);

        assertEq(l2CrossDomainMessenger.messageNonce(), nonce + 1);
    }

    /// @notice Tests that a message that was relayed here cannot be exported.
    function testFuzz_exportUndeliveredMessage_relayed_reverts(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        external
    {
        vm.assume(_source != block.chainid);
        address target = makeAddr("target");
        Identifier memory id = Identifier(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, 1, 1, 1, _source);
        bytes memory sentMessage = abi.encodePacked(
            abi.encode(SENT_MESSAGE_EVENT_SELECTOR, block.chainid, target, _nonce), // topics
            abi.encode(_sender, _message) // data
        );
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeCall(ICrossL2Inbox.validateMessage, (id, keccak256(sentMessage))),
            returnData: ""
        });
        messenger.relayMessage(id, sentMessage);

        vm.expectRevert(IUndeliveredMessageExporter.UndeliveredMessageExporter_MessageRelayed.selector);
        exporter.exportUndeliveredMessage(sourceMessenger, _source, _nonce, _sender, target, _message, 0);
    }

    /// @notice Tests that word from a chain other than the message's destination cannot expire it:
    ///         that chain hashes the message with itself as the destination.
    function test_exportUndeliveredMessage_otherChain_reverts() external {
        uint256 source = block.chainid;
        uint256 nonce = messenger.messageNonce();
        vm.prank(alice);
        bytes32 messageHash = messenger.sendMessage(source + 1, bob, hex"1234");

        vm.chainId(source + 1);
        bytes32 fromDestination =
            exporter.exportUndeliveredMessage(sourceMessenger, source, nonce, alice, bob, hex"1234", 0);
        vm.chainId(source + 2);
        bytes32 fromOther = exporter.exportUndeliveredMessage(sourceMessenger, source, nonce, alice, bob, hex"1234", 0);
        vm.chainId(source);

        assertEq(fromDestination, messageHash);
        assertNotEq(fromOther, messageHash);
        _expectExpireRejected(fromOther);
    }

    /// @notice Tests that word a chain exports about a message from itself can never expire one: no
    ///         chain can send a message to itself, so no such message was ever sent.
    function testFuzz_exportUndeliveredMessage_fromItself_reverts(
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message
    )
        external
    {
        bytes32 selfHash =
            exporter.exportUndeliveredMessage(sourceMessenger, block.chainid, _nonce, _sender, _target, _message, 0);

        _expectExpireRejected(selfHash);
    }

    /// @notice Expects this chain's messenger to reject word, from this chain's
    ///         L1CrossDomainMessenger, that a message it never sent was not relayed.
    function _expectExpireRejected(bytes32 _messageHash) internal {
        vm.mockCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l2CrossDomainMessenger.otherMessenger()))
        );
        vm.expectRevert(IL2ToL2CrossDomainMessenger.InvalidMessage.selector);
        vm.prank(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        messenger.expireMessage(_messageHash, type(uint64).max);
    }
}

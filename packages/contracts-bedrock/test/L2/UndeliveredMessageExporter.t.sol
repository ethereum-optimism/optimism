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
    event UndeliveredMessageExported(
        bytes32 indexed messageHash, uint256 indexed source, address sourceMessenger, uint256 undeliveredAt
    );

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
        _expectExport(messageHash, _source, _minGasLimit);

        bytes32 returned = exporter.exportUndeliveredMessage(
            sourceMessenger, _source, _nonce, _sender, _target, _message, _minGasLimit
        );

        assertEq(returned, messageHash);
    }

    /// @notice Tests that anyone can export, including an aliased L1 address sending a deposit.
    function testFuzz_exportUndeliveredMessage_anyCaller_succeeds(address _caller, bool _aliased) external {
        address caller = _aliased ? AddressAliasHelper.applyL1ToL2Alias(_caller) : _caller;
        bytes32 messageHash = Hashing.hashL2toL2CrossDomainMessage(block.chainid, 1, 0, alice, bob, hex"1234");
        _expectExport(messageHash, 1, 1_000_000);

        vm.prank(caller, caller);
        exporter.exportUndeliveredMessage(sourceMessenger, 1, 0, alice, bob, hex"1234", 1_000_000);
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
    function test_exportUndeliveredMessage_otherChainWordCannotExpire_succeeds() external {
        uint256 source = block.chainid;
        uint256 nonce = messenger.messageNonce();
        vm.prank(alice);
        bytes32 messageHash = messenger.sendMessage(source + 1, bob, hex"1234");

        vm.chainId(source + 1);
        _expectExport(messageHash, source, 0);
        exporter.exportUndeliveredMessage(sourceMessenger, source, nonce, alice, bob, hex"1234", 0);

        vm.chainId(source + 2);
        bytes32 fromOther = Hashing.hashL2toL2CrossDomainMessage(source + 2, source, nonce, alice, bob, hex"1234");
        assertNotEq(fromOther, messageHash);
        _expectExport(fromOther, source, 0);
        exporter.exportUndeliveredMessage(sourceMessenger, source, nonce, alice, bob, hex"1234", 0);

        vm.chainId(source);
        _expectExpireRejected(fromOther);
    }

    /// @notice Tests that word a chain exports about a message from itself can never expire one: the
    ///         hash names this chain as both source and destination, and no chain can send a message
    ///         to itself.
    function testFuzz_exportUndeliveredMessage_fromItselfWordCannotExpire_succeeds(
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message
    )
        external
    {
        bytes32 selfHash =
            Hashing.hashL2toL2CrossDomainMessage(block.chainid, block.chainid, _nonce, _sender, _target, _message);
        _expectExport(selfHash, block.chainid, 0);
        exporter.exportUndeliveredMessage(sourceMessenger, block.chainid, _nonce, _sender, _target, _message, 0);

        vm.expectRevert(IL2ToL2CrossDomainMessenger.MessageDestinationSameChain.selector);
        messenger.sendMessage(block.chainid, _target, _message);
        _expectExpireRejected(selfHash);
    }

    /// @notice Expects the exporter to send word that `_messageHash` was not relayed by now, as itself,
    ///         through the L2CrossDomainMessenger to the source messenger, and to emit
    ///         UndeliveredMessageExported.
    function _expectExport(bytes32 _messageHash, uint256 _source, uint32 _minGasLimit) internal {
        vm.expectEmit(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        emit SentMessage(
            sourceMessenger,
            Predeploys.UNDELIVERED_MESSAGE_EXPORTER,
            abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (_messageHash, block.timestamp)),
            l2CrossDomainMessenger.messageNonce(),
            _minGasLimit
        );
        vm.expectEmit(Predeploys.UNDELIVERED_MESSAGE_EXPORTER);
        emit UndeliveredMessageExported(_messageHash, _source, sourceMessenger, block.timestamp);
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

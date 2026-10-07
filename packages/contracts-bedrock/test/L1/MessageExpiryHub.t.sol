// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract
import { MessageExpiryHub } from "src/L1/MessageExpiryHub.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

/// @title MessageExpiryHub_TestInit
/// @notice Reusable test initialization for `MessageExpiryHub` tests. Mocks two chains of one
///         cluster, sharing an `ETHLockbox` and an `AnchorStateRegistry`: the destination of a
///         message, which reports it undelivered, and its source, which the fact is forwarded to.
abstract contract MessageExpiryHub_TestInit is Test {
    event UndeliveredMessageRecorded(
        bytes32 indexed cluster, bytes32 indexed messageHash, uint256 indexed source, uint256 undeliveredAt
    );
    event UndeliveredMessageForwarded(
        bytes32 indexed cluster, bytes32 indexed messageHash, uint256 indexed source, uint256 undeliveredAt
    );

    uint256 internal constant DESTINATION_CHAIN_ID = 901;
    uint256 internal constant SOURCE_CHAIN_ID = 902;
    bytes32 internal constant MESSAGE_HASH = keccak256("message");
    uint256 internal constant UNDELIVERED_AT = 1_000_000;
    uint32 internal constant MIN_GAS_LIMIT = 200_000;

    MessageExpiryHub internal hub;

    address internal destinationConfig;
    address internal destinationMessenger;
    address internal destinationPortal;
    address internal sourceConfig;
    address internal sourceMessenger;
    address internal sourcePortal;
    address internal lockbox;
    address internal asr;

    /// @notice Test setup.
    function setUp() public virtual {
        hub = new MessageExpiryHub();

        lockbox = _mockContract("lockbox");
        asr = makeAddr("asr");
        (destinationConfig, destinationMessenger, destinationPortal) =
            _mockChain("destination", DESTINATION_CHAIN_ID, lockbox, asr);
        (sourceConfig, sourceMessenger, sourcePortal) = _mockChain("source", SOURCE_CHAIN_ID, lockbox, asr);
    }

    /// @notice Creates a labelled address with code, so calls to it can be mocked.
    function _mockContract(string memory _name) internal returns (address addr_) {
        addr_ = makeAddr(_name);
        vm.etch(addr_, hex"01");
    }

    /// @notice Mocks a chain whose SystemConfig, L1CrossDomainMessenger and OptimismPortal point at
    ///         each other, and whose portal is authorized by `_lockbox`. Its messenger relays from
    ///         the L2ToL2CrossDomainMessenger.
    function _mockChain(
        string memory _name,
        uint256 _chainId,
        address _lockbox,
        address _asr
    )
        internal
        returns (address systemConfig_, address messenger_, address portal_)
    {
        systemConfig_ = _mockContract(string.concat(_name, "Config"));
        messenger_ = _mockContract(string.concat(_name, "Messenger"));
        portal_ = _mockContract(string.concat(_name, "Portal"));

        vm.mockCall(systemConfig_, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(messenger_));
        vm.mockCall(systemConfig_, abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(portal_));
        vm.mockCall(systemConfig_, abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(_chainId));
        vm.mockCall(messenger_, abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()), abi.encode(systemConfig_));
        vm.mockCall(
            messenger_,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        );
        vm.mockCall(portal_, abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(systemConfig_));
        vm.mockCall(portal_, abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(_lockbox));
        vm.mockCall(portal_, abi.encodeCall(IOptimismPortal2.anchorStateRegistry, ()), abi.encode(_asr));
        vm.mockCall(
            _lockbox,
            abi.encodeCall(IETHLockbox.authorizedPortals, (IOptimismPortal2(payable(portal_)))),
            abi.encode(true)
        );
    }

    /// @notice The cluster of the mocked chains.
    function _cluster() internal view returns (bytes32) {
        return keccak256(abi.encode(lockbox, asr));
    }

    /// @notice Records the fact as the destination's messenger relaying it.
    function _receive() internal {
        vm.prank(destinationMessenger);
        hub.receiveUndeliveredMessage(MESSAGE_HASH, SOURCE_CHAIN_ID, UNDELIVERED_AT);
    }

    /// @notice The call the hub makes on the source's messenger to forward the fact.
    function _forwardCall() internal pure returns (bytes memory) {
        return abi.encodeCall(
            ICrossDomainMessenger.sendMessage,
            (
                Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
                abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (MESSAGE_HASH, UNDELIVERED_AT)),
                MIN_GAS_LIMIT
            )
        );
    }
}

/// @title MessageExpiryHub_ReceiveUndeliveredMessage_Test
/// @notice Tests the `receiveUndeliveredMessage` function of the `MessageExpiryHub` contract.
contract MessageExpiryHub_ReceiveUndeliveredMessage_Test is MessageExpiryHub_TestInit {
    /// @notice Tests that a fact relayed by a chain's messenger from its L2ToL2CrossDomainMessenger
    ///         is recorded under the chain's cluster.
    function test_receiveUndeliveredMessage_succeeds() external {
        bytes32 id = hub.factId(_cluster(), MESSAGE_HASH, SOURCE_CHAIN_ID, UNDELIVERED_AT);
        assertEq(hub.cluster(ISystemConfig(destinationConfig)), _cluster());

        vm.expectEmit(address(hub));
        emit UndeliveredMessageRecorded(_cluster(), MESSAGE_HASH, SOURCE_CHAIN_ID, UNDELIVERED_AT);
        _receive();

        assertTrue(hub.facts(id));
    }

    /// @notice Tests that a fact sent through the messenger by anything but the
    ///         L2ToL2CrossDomainMessenger is rejected.
    function testFuzz_receiveUndeliveredMessage_wrongCrossDomainSender_reverts(address _xDomainSender) external {
        vm.assume(_xDomainSender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vm.mockCall(
            destinationMessenger,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(_xDomainSender)
        );

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidSender.selector);
        _receive();
    }

    /// @notice Tests that a caller that names a real chain's SystemConfig, but is not that chain's
    ///         messenger, is rejected.
    function test_receiveUndeliveredMessage_notTheChainsMessenger_reverts() external {
        address impostor = _mockContract("impostor");
        vm.mockCall(impostor, abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()), abi.encode(destinationConfig));
        vm.mockCall(
            impostor,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        );

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidSender.selector);
        vm.prank(impostor);
        hub.receiveUndeliveredMessage(MESSAGE_HASH, SOURCE_CHAIN_ID, UNDELIVERED_AT);
    }

    /// @notice Tests that a forged chain that borrows a real portal is rejected: the real portal
    ///         points back only at its real SystemConfig.
    function test_receiveUndeliveredMessage_borrowedPortal_reverts() external {
        (address fakeConfig, address fakeMessenger,) = _mockChain("fake", DESTINATION_CHAIN_ID, lockbox, asr);
        vm.mockCall(fakeConfig, abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(destinationPortal));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidChain.selector);
        vm.prank(fakeMessenger);
        hub.receiveUndeliveredMessage(MESSAGE_HASH, SOURCE_CHAIN_ID, UNDELIVERED_AT);
    }

    /// @notice Tests that a forged chain whose portal the lockbox does not authorize is rejected.
    function test_receiveUndeliveredMessage_unauthorizedPortal_reverts() external {
        (, address fakeMessenger, address fakePortal) = _mockChain("fake", DESTINATION_CHAIN_ID, lockbox, asr);
        vm.mockCall(
            lockbox,
            abi.encodeCall(IETHLockbox.authorizedPortals, (IOptimismPortal2(payable(fakePortal)))),
            abi.encode(false)
        );

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidChain.selector);
        vm.prank(fakeMessenger);
        hub.receiveUndeliveredMessage(MESSAGE_HASH, SOURCE_CHAIN_ID, UNDELIVERED_AT);
    }

    /// @notice Tests that a SystemConfig with a messenger or portal that does not point back at it,
    ///         or with no messenger, portal or lockbox, is rejected.
    function test_receiveUndeliveredMessage_unboundChain_reverts() external {
        vm.mockCall(
            destinationMessenger, abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()), abi.encode(sourceConfig)
        );
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidChain.selector);
        hub.cluster(ISystemConfig(destinationConfig));

        vm.mockCall(destinationConfig, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(address(0)));
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidChain.selector);
        hub.cluster(ISystemConfig(destinationConfig));

        vm.mockCall(sourcePortal, abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(destinationConfig));
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidChain.selector);
        hub.cluster(ISystemConfig(sourceConfig));

        vm.mockCall(sourceConfig, abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(address(0)));
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidChain.selector);
        hub.cluster(ISystemConfig(sourceConfig));

        (address otherConfig,, address otherPortal) = _mockChain("other", 903, lockbox, asr);
        vm.mockCall(otherPortal, abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(address(0)));
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidChain.selector);
        hub.cluster(ISystemConfig(otherConfig));
    }
}

/// @title MessageExpiryHub_ForwardUndeliveredMessage_Test
/// @notice Tests the `forwardUndeliveredMessage` function of the `MessageExpiryHub` contract.
contract MessageExpiryHub_ForwardUndeliveredMessage_Test is MessageExpiryHub_TestInit {
    /// @notice Tests that a recorded fact is forwarded to the source's L2ToL2CrossDomainMessenger
    ///         through the source's messenger, and can be forwarded again.
    function test_forwardUndeliveredMessage_succeeds() external {
        _receive();
        vm.mockCall(sourceMessenger, _forwardCall(), "");

        for (uint256 i; i < 2; i++) {
            vm.expectCall(sourceMessenger, _forwardCall());
            vm.expectEmit(address(hub));
            emit UndeliveredMessageForwarded(_cluster(), MESSAGE_HASH, SOURCE_CHAIN_ID, UNDELIVERED_AT);
            hub.forwardUndeliveredMessage(ISystemConfig(sourceConfig), MESSAGE_HASH, UNDELIVERED_AT, MIN_GAS_LIMIT);
        }
    }

    /// @notice Tests that a fact that was never recorded, or is forwarded with a different
    ///         timestamp, cannot be forwarded.
    function testFuzz_forwardUndeliveredMessage_unknownFact_reverts(uint256 _undeliveredAt) external {
        vm.assume(_undeliveredAt != UNDELIVERED_AT);
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnknownFact.selector);
        hub.forwardUndeliveredMessage(ISystemConfig(sourceConfig), MESSAGE_HASH, UNDELIVERED_AT, MIN_GAS_LIMIT);

        _receive();
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnknownFact.selector);
        hub.forwardUndeliveredMessage(ISystemConfig(sourceConfig), MESSAGE_HASH, _undeliveredAt, MIN_GAS_LIMIT);
    }

    /// @notice Tests that a fact cannot be forwarded to a chain of the cluster other than the
    ///         source it names.
    function test_forwardUndeliveredMessage_otherChain_reverts() external {
        _receive();
        (address otherConfig,,) = _mockChain("other", 903, lockbox, asr);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnknownFact.selector);
        hub.forwardUndeliveredMessage(ISystemConfig(otherConfig), MESSAGE_HASH, UNDELIVERED_AT, MIN_GAS_LIMIT);
    }

    /// @notice Tests that a fact cannot be forwarded to a chain with the source's chain ID in
    ///         another cluster: one with a different lockbox, or a different AnchorStateRegistry.
    function test_forwardUndeliveredMessage_otherCluster_reverts() external {
        _receive();
        address otherLockbox = _mockContract("otherLockbox");
        (address otherLockboxConfig,,) = _mockChain("otherLockboxSource", SOURCE_CHAIN_ID, otherLockbox, asr);
        (address otherAsrConfig,,) = _mockChain("otherAsrSource", SOURCE_CHAIN_ID, lockbox, makeAddr("otherAsr"));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnknownFact.selector);
        hub.forwardUndeliveredMessage(ISystemConfig(otherLockboxConfig), MESSAGE_HASH, UNDELIVERED_AT, MIN_GAS_LIMIT);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnknownFact.selector);
        hub.forwardUndeliveredMessage(ISystemConfig(otherAsrConfig), MESSAGE_HASH, UNDELIVERED_AT, MIN_GAS_LIMIT);
    }

    /// @notice Tests that a fact recorded by a forged cluster, with its own lockbox, is isolated
    ///         from the real cluster.
    function test_forwardUndeliveredMessage_forgedCluster_reverts() external {
        address fakeLockbox = _mockContract("fakeLockbox");
        (, address fakeMessenger,) = _mockChain("fake", DESTINATION_CHAIN_ID, fakeLockbox, asr);
        vm.prank(fakeMessenger);
        hub.receiveUndeliveredMessage(MESSAGE_HASH, SOURCE_CHAIN_ID, UNDELIVERED_AT);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnknownFact.selector);
        hub.forwardUndeliveredMessage(ISystemConfig(sourceConfig), MESSAGE_HASH, UNDELIVERED_AT, MIN_GAS_LIMIT);
    }

    /// @notice Tests that a forged source SystemConfig that borrows the real source's messenger is
    ///         rejected, so a fact cannot be forwarded through a messenger it does not own.
    function test_forwardUndeliveredMessage_forgedSource_reverts() external {
        _receive();
        address fakeConfig = _mockContract("fakeConfig");
        vm.mockCall(fakeConfig, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(sourceMessenger));
        vm.mockCall(fakeConfig, abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(sourcePortal));
        vm.mockCall(fakeConfig, abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(SOURCE_CHAIN_ID));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidChain.selector);
        hub.forwardUndeliveredMessage(ISystemConfig(fakeConfig), MESSAGE_HASH, UNDELIVERED_AT, MIN_GAS_LIMIT);
    }
}

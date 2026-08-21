// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { MockHelper } from "test/utils/MockHelper.sol";

// Target contract
import { MessageExpiryHub } from "src/L1/MessageExpiryHub.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IMessageExpiryRelay } from "interfaces/L2/IMessageExpiryRelay.sol";

/// @title MessageExpiryHub_TestInit
/// @notice Reusable test initialization for `MessageExpiryHub` tests. Builds two mocked chains that
///         share an `ETHLockbox` and an `AnchorStateRegistry`, i.e. one interop cluster: the
///         attestor chain (which sends notices to the hub) and the source chain (which notices are
///         forwarded to).
abstract contract MessageExpiryHub_TestInit is Test, MockHelper {
    event ChainRegistered(address indexed ethLockbox, uint256 indexed chainId, address systemConfig);

    event ExpiryNoticeReceived(
        address indexed ethLockbox,
        uint256 indexed attestorChainId,
        bytes32 indexed msgHash,
        uint256 sourceChainId,
        uint256 attestedAt
    );

    event ExpiryNoticeForwarded(
        address indexed ethLockbox, uint256 indexed attestorChainId, bytes32 indexed msgHash, uint256 sourceChainId
    );

    /// @notice Address of the `MessageExpiryRelay` predeploy, mirroring the hub's own constant.
    address internal constant MESSAGE_EXPIRY_RELAY = 0x420000000000000000000000000000000000002E;

    /// @notice Chain ID of the attesting (destination) chain.
    uint256 internal constant ATTESTOR_CHAIN_ID = 901;

    /// @notice Chain ID of the source chain that notices are forwarded to.
    uint256 internal constant SOURCE_CHAIN_ID = 902;

    /// @notice Minimum gas limit used for cross domain messages.
    uint32 internal constant MIN_GAS_LIMIT = 500_000;

    /// @notice The `MessageExpiryHub` under test.
    MessageExpiryHub internal messageExpiryHub;

    /// @notice `SystemConfig` of the attestor chain.
    address internal systemConfig;

    /// @notice `L1CrossDomainMessenger` of the attestor chain.
    address internal messenger;

    /// @notice `OptimismPortal2` of the attestor chain.
    address internal portal;

    /// @notice `SystemConfig` of the source chain.
    address internal systemConfig2;

    /// @notice `L1CrossDomainMessenger` of the source chain.
    address internal messenger2;

    /// @notice `OptimismPortal2` of the source chain.
    address internal portal2;

    /// @notice `ETHLockbox` shared by both chains, i.e. the cluster identity.
    address internal lockbox;

    /// @notice `AnchorStateRegistry` shared by both chains.
    address internal asr;

    /// @notice Test setup.
    function setUp() public virtual {
        // Move to a realistic L1 timestamp so that the fixed attestation timestamps used across
        // these tests are in the past, satisfying the hub's `attestedAt <= block.timestamp` bound.
        vm.warp(1_000_000_000);

        messageExpiryHub = new MessageExpiryHub();

        systemConfig = _makeMockContract("systemConfig");
        messenger = _makeMockContract("messenger");
        portal = _makeMockContract("portal");
        systemConfig2 = _makeMockContract("systemConfig2");
        messenger2 = _makeMockContract("messenger2");
        portal2 = _makeMockContract("portal2");
        lockbox = _makeMockContract("lockbox");
        asr = makeAddr("asr");

        _mockChain(systemConfig, messenger, portal, ATTESTOR_CHAIN_ID);
        _mockChain(systemConfig2, messenger2, portal2, SOURCE_CHAIN_ID);
        _mockCluster(portal, lockbox, asr);
        _mockCluster(portal2, lockbox, asr);
        _mockXDomainMessageSender(messenger, MESSAGE_EXPIRY_RELAY);
        _mockXDomainMessageSender(messenger2, MESSAGE_EXPIRY_RELAY);
    }

    /// @notice Creates a labelled address with dummy code so that calls to it can be mocked.
    /// @param _name Label for the address.
    /// @return addr_ The address.
    function _makeMockContract(string memory _name) internal returns (address addr_) {
        addr_ = makeAddr(_name);
        vm.etch(addr_, hex"01");
    }

    /// @notice Mocks the identity of a chain: a `SystemConfig` bound to its messenger and portal,
    ///         with the portal bound back to the `SystemConfig`.
    /// @param _systemConfig `SystemConfig` of the chain.
    /// @param _messenger    `L1CrossDomainMessenger` of the chain.
    /// @param _portal       `OptimismPortal2` of the chain.
    /// @param _chainId      Chain ID of the chain.
    function _mockChain(address _systemConfig, address _messenger, address _portal, uint256 _chainId) internal {
        vm.mockCall(_systemConfig, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(_messenger));
        vm.mockCall(_messenger, abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()), abi.encode(_systemConfig));
        vm.mockCall(_systemConfig, abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(_portal));
        vm.mockCall(_portal, abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(_systemConfig));
        vm.mockCall(_systemConfig, abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(_chainId));
    }

    /// @notice Mocks the cluster membership of a chain's portal.
    /// @param _portal              `OptimismPortal2` of the chain.
    /// @param _lockbox             Shared `ETHLockbox` authorizing the portal.
    /// @param _anchorStateRegistry Shared `AnchorStateRegistry` of the cluster.
    function _mockCluster(address _portal, address _lockbox, address _anchorStateRegistry) internal {
        vm.mockCall(_portal, abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(_lockbox));
        vm.mockCall(_portal, abi.encodeCall(IOptimismPortal2.anchorStateRegistry, ()), abi.encode(_anchorStateRegistry));
        vm.mockCall(
            _lockbox,
            abi.encodeCall(IETHLockbox.authorizedPortals, (IOptimismPortal2(payable(_portal)))),
            abi.encode(true)
        );
    }

    /// @notice Mocks the cross domain sender reported by a messenger.
    /// @param _messenger Messenger to mock.
    /// @param _sender    Cross domain sender to report.
    function _mockXDomainMessageSender(address _messenger, address _sender) internal {
        vm.mockCall(_messenger, abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()), abi.encode(_sender));
    }

    /// @notice Delivers an expiry notice to the hub as if relayed by the attestor chain's messenger.
    /// @param _msgHash       Hash of the attested message.
    /// @param _sourceChainId Chain ID the message was sent from.
    /// @param _attestedAt    Timestamp of the attestation.
    function _receiveNotice(bytes32 _msgHash, uint256 _sourceChainId, uint256 _attestedAt) internal {
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(_msgHash, _sourceChainId, _attestedAt);
    }
}

/// @title MessageExpiryHub_RegisterChain_Test
/// @notice Tests the `registerChain` function of the `MessageExpiryHub` contract.
contract MessageExpiryHub_RegisterChain_Test is MessageExpiryHub_TestInit {
    /// @notice Tests that a chain of a shared lockbox cluster can be registered.
    function test_registerChain_succeeds() public {
        vm.expectEmit(address(messageExpiryHub));
        emit ChainRegistered(lockbox, ATTESTOR_CHAIN_ID, systemConfig);

        messageExpiryHub.registerChain(ISystemConfig(systemConfig));

        assertEq(address(messageExpiryHub.registeredChains(lockbox, ATTESTOR_CHAIN_ID)), systemConfig);
    }

    /// @notice Tests that registering the same chain again simply re-validates it.
    function test_registerChain_reRegistration_succeeds() public {
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));

        assertEq(address(messageExpiryHub.registeredChains(lockbox, ATTESTOR_CHAIN_ID)), systemConfig);
    }

    /// @notice Tests that registering reverts when the `SystemConfig` has no messenger.
    function test_registerChain_zeroMessenger_reverts() public {
        vm.mockCall(systemConfig, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(address(0)));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidMessenger.selector);
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));
    }

    /// @notice Tests that registering reverts when the `SystemConfig` and its messenger are not
    ///         bound to each other.
    function test_registerChain_messengerMismatch_reverts() public {
        vm.mockCall(
            messenger, abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()), abi.encode(makeAddr("otherConfig"))
        );

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidMessenger.selector);
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));
    }

    /// @notice Tests that registering reverts when the chain has no portal.
    function test_registerChain_zeroPortal_reverts() public {
        vm.mockCall(systemConfig, abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(address(0)));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));
    }

    /// @notice Tests that registering reverts when the chain's portal is not bound back to the
    ///         `SystemConfig` that named it, i.e. the reverse portal binding fails.
    function test_registerChain_portalNotBound_reverts() public {
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(makeAddr("otherConfig")));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));
    }

    /// @notice Tests that a forged `SystemConfig` cannot register by borrowing another chain's real,
    ///         cluster-authorized portal: the reverse portal binding rejects it because the real
    ///         portal points back at its own `SystemConfig`, not the forgery.
    function test_registerChain_borrowedPortal_reverts() public {
        address fakeSystemConfig = _makeMockContract("fakeSystemConfig");
        address fakeMessenger = _makeMockContract("fakeMessenger");

        vm.mockCall(
            fakeSystemConfig, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(fakeMessenger)
        );
        vm.mockCall(
            fakeMessenger, abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()), abi.encode(fakeSystemConfig)
        );
        // Borrow the attestor chain's real portal, which is authorized by the real lockbox.
        vm.mockCall(fakeSystemConfig, abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(portal));
        vm.mockCall(fakeSystemConfig, abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(uint256(66_666)));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        messageExpiryHub.registerChain(ISystemConfig(fakeSystemConfig));
    }

    /// @notice Tests that registering reverts when the chain's portal has no lockbox.
    function test_registerChain_zeroLockbox_reverts() public {
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(address(0)));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));
    }

    /// @notice Tests that registering reverts when the lockbox does not authorize the portal.
    function test_registerChain_unauthorizedPortal_reverts() public {
        vm.mockCall(
            lockbox,
            abi.encodeCall(IETHLockbox.authorizedPortals, (IOptimismPortal2(payable(portal)))),
            abi.encode(false)
        );

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));
    }

    /// @notice Tests that registering reverts when the chain reports a zero chain ID.
    function test_registerChain_zeroChainId_reverts() public {
        vm.mockCall(systemConfig, abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(uint256(0)));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidChainId.selector);
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));
    }

    /// @notice Tests that an existing registration cannot be overwritten with a different
    ///         `SystemConfig` for the same cluster and chain ID, while re-registering the original
    ///         stays idempotent.
    function test_registerChain_overwriteDifferentSystemConfig_reverts() public {
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));

        // A different, independently valid `SystemConfig` in the same cluster that reports the same
        // chain ID.
        address systemConfigDup = _makeMockContract("systemConfigDup");
        address messengerDup = _makeMockContract("messengerDup");
        address portalDup = _makeMockContract("portalDup");
        _mockChain(systemConfigDup, messengerDup, portalDup, ATTESTOR_CHAIN_ID);
        _mockCluster(portalDup, lockbox, asr);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_AlreadyRegistered.selector);
        messageExpiryHub.registerChain(ISystemConfig(systemConfigDup));

        // The original registration is intact and re-registering it still succeeds.
        assertEq(address(messageExpiryHub.registeredChains(lockbox, ATTESTOR_CHAIN_ID)), systemConfig);
        messageExpiryHub.registerChain(ISystemConfig(systemConfig));
        assertEq(address(messageExpiryHub.registeredChains(lockbox, ATTESTOR_CHAIN_ID)), systemConfig);
    }
}

/// @title MessageExpiryHub_ReceiveExpiryNotice_Test
/// @notice Tests the `receiveExpiryNotice` function of the `MessageExpiryHub` contract.
contract MessageExpiryHub_ReceiveExpiryNotice_Test is MessageExpiryHub_TestInit {
    /// @notice Hash of the attested message.
    bytes32 internal msgHash = keccak256("undelivered message");

    /// @notice Timestamp of the attestation.
    uint256 internal constant ATTESTED_AT = 1000;

    /// @notice Tests that a notice relayed by a cluster chain's messenger is recorded under that
    ///         cluster's lockbox and the message's source chain.
    function test_receiveExpiryNotice_succeeds() public {
        vm.expectEmit(address(messageExpiryHub));
        emit ExpiryNoticeReceived(lockbox, ATTESTOR_CHAIN_ID, msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);

        _receiveNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);

        (address anchorStateRegistry, uint64 attestedAt) =
            messageExpiryHub.notices(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash);
        assertEq(anchorStateRegistry, asr);
        assertEq(uint256(attestedAt), ATTESTED_AT);
    }

    /// @notice Tests that a notice is rejected when the calling messenger and its `SystemConfig`
    ///         are not bound to each other.
    function test_receiveExpiryNotice_messengerMismatch_reverts() public {
        vm.mockCall(
            systemConfig, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(makeAddr("other"))
        );

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidMessenger.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);
    }

    /// @notice Tests that a notice is rejected when the cross domain sender is not the
    ///         `MessageExpiryRelay` predeploy.
    /// @param _sender Cross domain sender of the relayed message.
    function testFuzz_receiveExpiryNotice_invalidCrossDomainSender_reverts(address _sender) public {
        vm.assume(_sender != MESSAGE_EXPIRY_RELAY);
        _mockXDomainMessageSender(messenger, _sender);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidCrossDomainSender.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);
    }

    /// @notice Tests that a notice is rejected when the attestor chain has no portal.
    function test_receiveExpiryNotice_zeroPortal_reverts() public {
        vm.mockCall(systemConfig, abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(address(0)));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);
    }

    /// @notice Tests that a notice is rejected when the attestor's portal is not bound back to its
    ///         `SystemConfig`, i.e. the reverse portal binding fails.
    function test_receiveExpiryNotice_portalNotBound_reverts() public {
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(makeAddr("otherConfig")));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);
    }

    /// @notice Tests that a forged `SystemConfig` cannot forge a notice by borrowing another chain's
    ///         real, cluster-authorized portal. This is the core C1 attestation-forgery regression:
    ///         the fake `SystemConfig` returns the real attestor portal and an attacker chain ID,
    ///         but the reverse portal binding rejects it because the real portal points back at its
    ///         own `SystemConfig`, so no forged notice is ever recorded under the real lockbox key.
    function test_receiveExpiryNotice_borrowedPortal_reverts() public {
        address fakeSystemConfig = _makeMockContract("fakeSystemConfig");
        address fakeMessenger = _makeMockContract("fakeMessenger");
        uint256 attackerChainId = 66_666;

        vm.mockCall(
            fakeSystemConfig, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(fakeMessenger)
        );
        vm.mockCall(
            fakeMessenger, abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()), abi.encode(fakeSystemConfig)
        );
        // Borrow the attestor chain's real, cluster-authorized portal.
        vm.mockCall(fakeSystemConfig, abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(portal));
        vm.mockCall(fakeSystemConfig, abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(attackerChainId));
        _mockXDomainMessageSender(fakeMessenger, MESSAGE_EXPIRY_RELAY);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        vm.prank(fakeMessenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);

        // Nothing was recorded under the real cluster's lockbox key for the attacker chain ID.
        (, uint64 attestedAt) = messageExpiryHub.notices(lockbox, attackerChainId, SOURCE_CHAIN_ID, msgHash);
        assertEq(uint256(attestedAt), 0);
    }

    /// @notice Tests that a notice is rejected when the attestor's portal has no lockbox.
    function test_receiveExpiryNotice_zeroLockbox_reverts() public {
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(address(0)));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);
    }

    /// @notice Tests that a notice is rejected when the attestor's portal is not authorized by its
    ///         lockbox.
    function test_receiveExpiryNotice_unauthorizedPortal_reverts() public {
        vm.mockCall(
            lockbox,
            abi.encodeCall(IETHLockbox.authorizedPortals, (IOptimismPortal2(payable(portal)))),
            abi.encode(false)
        );

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);
    }

    /// @notice Tests that a notice is rejected when the attestor reports a zero chain ID.
    function test_receiveExpiryNotice_zeroChainId_reverts() public {
        vm.mockCall(systemConfig, abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(uint256(0)));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidChainId.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);
    }

    /// @notice Tests that a notice naming the attestor itself as the message source is rejected.
    function test_receiveExpiryNotice_invalidSourceChain_reverts() public {
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidSourceChain.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, ATTESTOR_CHAIN_ID, ATTESTED_AT);
    }

    /// @notice Tests that a notice whose attestation timestamp does not fit in a `uint64`, which is
    ///         necessarily in the future, is rejected.
    /// @param _attestedAt Timestamp of the attestation.
    function testFuzz_receiveExpiryNotice_invalidTimestamp_reverts(uint256 _attestedAt) public {
        _attestedAt = bound(_attestedAt, uint256(type(uint64).max) + 1, type(uint256).max);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidTimestamp.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, _attestedAt);
    }

    /// @notice Tests that a future-dated attestation (after the current L1 timestamp) is rejected,
    ///         removing the unbounded-future-timestamp lever.
    function test_receiveExpiryNotice_futureTimestamp_reverts() public {
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidTimestamp.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, block.timestamp + 1);
    }

    /// @notice Tests that an attestation taken exactly at the current L1 timestamp is accepted.
    function test_receiveExpiryNotice_currentTimestamp_succeeds() public {
        _receiveNotice(msgHash, SOURCE_CHAIN_ID, block.timestamp);

        (, uint64 attestedAt) = messageExpiryHub.notices(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash);
        assertEq(uint256(attestedAt), block.timestamp);
    }

    /// @notice Tests that a notice with the same attestation timestamp as the stored one is
    ///         rejected.
    function test_receiveExpiryNotice_equalTimestamp_reverts() public {
        _receiveNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_StaleNotice.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);
    }

    /// @notice Tests that a notice older than the stored one is rejected.
    function test_receiveExpiryNotice_staleNotice_reverts() public {
        _receiveNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_StaleNotice.selector);
        vm.prank(messenger);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT - 1);
    }

    /// @notice Tests that a strictly newer notice for the same key supersedes the stored one.
    function test_receiveExpiryNotice_newerNotice_succeeds() public {
        _receiveNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);

        // The attestor keeps its lockbox and source chain, i.e. the notice key, but moves to a
        // different registry before the second attestation, so the overwrite is observable.
        address newAsr = makeAddr("newAsr");
        _mockCluster(portal, lockbox, newAsr);

        vm.expectEmit(address(messageExpiryHub));
        emit ExpiryNoticeReceived(lockbox, ATTESTOR_CHAIN_ID, msgHash, SOURCE_CHAIN_ID, ATTESTED_AT + 1);

        _receiveNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT + 1);

        (address anchorStateRegistry, uint64 attestedAt) =
            messageExpiryHub.notices(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash);
        assertEq(anchorStateRegistry, newAsr);
        assertEq(uint256(attestedAt), ATTESTED_AT + 1);
    }

    /// @notice Tests that two attestations for the same (lockbox, attestor, message) but different
    ///         source chains coexist under distinct keys: an attestation naming a bogus source chain
    ///         (even with a newer timestamp) cannot displace a pending legitimate notice.
    function test_receiveExpiryNotice_differentSourceChainId_coexist_succeeds() public {
        // A legitimate notice naming the message's true source chain.
        _receiveNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);

        // A later attestation for the same message but a different (bogus) source chain, with a
        // newer timestamp. It is stored under its own source-chain key and does not touch the
        // legitimate notice, since staleness is scoped per source chain.
        uint256 bogusSource = SOURCE_CHAIN_ID + 12_345;
        _receiveNotice(msgHash, bogusSource, ATTESTED_AT + 100);

        (address asrLegit, uint64 attestedAtLegit) =
            messageExpiryHub.notices(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash);
        assertEq(asrLegit, asr);
        assertEq(uint256(attestedAtLegit), ATTESTED_AT);

        (address asrBogus, uint64 attestedAtBogus) =
            messageExpiryHub.notices(lockbox, ATTESTOR_CHAIN_ID, bogusSource, msgHash);
        assertEq(asrBogus, asr);
        assertEq(uint256(attestedAtBogus), ATTESTED_AT + 100);
    }

    /// @notice Tests that a chain of a different cluster with a colliding chain ID cannot clobber
    ///         another cluster's notice for the same message hash: notices are keyed by the
    ///         attestor's shared lockbox, so both coexist and staleness is scoped to one cluster.
    function test_receiveExpiryNotice_collidingChainIdOtherCluster_succeeds() public {
        _receiveNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);

        // A second cluster whose attestor chain reports the very same chain ID.
        address systemConfigB = _makeMockContract("systemConfigB");
        address messengerB = _makeMockContract("messengerB");
        address portalB = _makeMockContract("portalB");
        address lockboxB = _makeMockContract("lockboxB");
        address asrB = makeAddr("asrB");
        _mockChain(systemConfigB, messengerB, portalB, ATTESTOR_CHAIN_ID);
        _mockCluster(portalB, lockboxB, asrB);
        _mockXDomainMessageSender(messengerB, MESSAGE_EXPIRY_RELAY);

        // An older attestation from the other cluster, for the same message and source chain, is
        // not stale: staleness is compared against that cluster's own (empty) slot.
        vm.expectEmit(address(messageExpiryHub));
        emit ExpiryNoticeReceived(lockboxB, ATTESTOR_CHAIN_ID, msgHash, SOURCE_CHAIN_ID, ATTESTED_AT - 1);
        vm.prank(messengerB);
        messageExpiryHub.receiveExpiryNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT - 1);

        // The first cluster's notice is untouched under its own lockbox key.
        (address anchorStateRegistryA, uint64 attestedAtA) =
            messageExpiryHub.notices(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash);
        assertEq(anchorStateRegistryA, asr);
        assertEq(uint256(attestedAtA), ATTESTED_AT);

        // The second cluster's notice lives under its own lockbox key.
        (address anchorStateRegistryB, uint64 attestedAtB) =
            messageExpiryHub.notices(lockboxB, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash);
        assertEq(anchorStateRegistryB, asrB);
        assertEq(uint256(attestedAtB), ATTESTED_AT - 1);
    }
}

/// @title MessageExpiryHub_ForwardExpiryNotice_Test
/// @notice Tests the `forwardExpiryNotice` function of the `MessageExpiryHub` contract.
contract MessageExpiryHub_ForwardExpiryNotice_Test is MessageExpiryHub_TestInit {
    /// @notice Hash of the attested message.
    bytes32 internal msgHash = keccak256("undelivered message");

    /// @notice Timestamp of the attestation.
    uint256 internal constant ATTESTED_AT = 1000;

    /// @notice Test setup. Registers the source chain and records a notice for it.
    function setUp() public virtual override {
        super.setUp();

        messageExpiryHub.registerChain(ISystemConfig(systemConfig2));
        _receiveNotice(msgHash, SOURCE_CHAIN_ID, ATTESTED_AT);
    }

    /// @notice Expected calldata of the forwarded message on the source chain.
    /// @return Encoded `receiveExpiry` call.
    function _expectedForward() internal view returns (bytes memory) {
        return abi.encodeCall(
            ICrossDomainMessenger.sendMessage,
            (
                MESSAGE_EXPIRY_RELAY,
                abi.encodeCall(IMessageExpiryRelay.receiveExpiry, (msgHash, ATTESTOR_CHAIN_ID, ATTESTED_AT)),
                MIN_GAS_LIMIT
            )
        );
    }

    /// @notice Tests that a recorded notice is forwarded to the source chain's messenger.
    function test_forwardExpiryNotice_succeeds() public {
        _mockAndExpect(messenger2, _expectedForward(), abi.encode());

        vm.expectEmit(address(messageExpiryHub));
        emit ExpiryNoticeForwarded(lockbox, ATTESTOR_CHAIN_ID, msgHash, SOURCE_CHAIN_ID);

        messageExpiryHub.forwardExpiryNotice(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash, MIN_GAS_LIMIT);
    }

    /// @notice Tests that forwarding is repeatable.
    function test_forwardExpiryNotice_repeated_succeeds() public {
        vm.mockCall(messenger2, _expectedForward(), abi.encode());

        messageExpiryHub.forwardExpiryNotice(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash, MIN_GAS_LIMIT);

        vm.expectCall(messenger2, _expectedForward());
        messageExpiryHub.forwardExpiryNotice(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash, MIN_GAS_LIMIT);
    }

    /// @notice Tests that forwarding a notice that was never recorded is rejected.
    function test_forwardExpiryNotice_noticeNotFound_reverts() public {
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_NoticeNotFound.selector);
        messageExpiryHub.forwardExpiryNotice(
            lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, keccak256("unknown"), MIN_GAS_LIMIT
        );
    }

    /// @notice Tests that a notice cannot be reached through the wrong source chain key.
    function test_forwardExpiryNotice_wrongSourceChain_reverts() public {
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_NoticeNotFound.selector);
        messageExpiryHub.forwardExpiryNotice(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID + 1, msgHash, MIN_GAS_LIMIT);
    }

    /// @notice Tests that a notice cannot be reached through another cluster's lockbox key.
    function test_forwardExpiryNotice_otherLockbox_reverts() public {
        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_NoticeNotFound.selector);
        messageExpiryHub.forwardExpiryNotice(
            _makeMockContract("otherLockbox"), ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash, MIN_GAS_LIMIT
        );
    }

    /// @notice Tests that forwarding to a source chain that was never registered is rejected.
    function test_forwardExpiryNotice_chainNotRegistered_reverts() public {
        bytes32 otherHash = keccak256("other message");
        _receiveNotice(otherHash, SOURCE_CHAIN_ID + 100, ATTESTED_AT);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_ChainNotRegistered.selector);
        messageExpiryHub.forwardExpiryNotice(
            lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID + 100, otherHash, MIN_GAS_LIMIT
        );
    }

    /// @notice Tests that forwarding is rejected when the source chain's messenger binding broke
    ///         after registration.
    function test_forwardExpiryNotice_messengerMismatch_reverts() public {
        vm.mockCall(
            messenger2, abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()), abi.encode(makeAddr("otherConfig"))
        );

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidMessenger.selector);
        messageExpiryHub.forwardExpiryNotice(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash, MIN_GAS_LIMIT);
    }

    /// @notice Tests that forwarding is rejected when the source chain's messenger was unset after
    ///         registration.
    function test_forwardExpiryNotice_zeroMessenger_reverts() public {
        vm.mockCall(systemConfig2, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(address(0)));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_InvalidMessenger.selector);
        messageExpiryHub.forwardExpiryNotice(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash, MIN_GAS_LIMIT);
    }

    /// @notice Tests that forwarding is rejected when the source chain's portal moved to a
    ///         different lockbox, i.e. a different cluster than the notice's.
    function test_forwardExpiryNotice_lockboxChanged_reverts() public {
        _mockCluster(portal2, _makeMockContract("otherLockbox"), asr);

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_ClusterMismatch.selector);
        messageExpiryHub.forwardExpiryNotice(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash, MIN_GAS_LIMIT);
    }

    /// @notice Tests that forwarding is rejected when the source chain no longer shares the
    ///         attestor's `AnchorStateRegistry`.
    function test_forwardExpiryNotice_anchorStateRegistryChanged_reverts() public {
        _mockCluster(portal2, lockbox, makeAddr("otherAsr"));

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_ClusterMismatch.selector);
        messageExpiryHub.forwardExpiryNotice(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash, MIN_GAS_LIMIT);
    }

    /// @notice Tests that forwarding is rejected when the source chain's portal lost its lockbox
    ///         authorization after registration.
    function test_forwardExpiryNotice_unauthorizedPortal_reverts() public {
        vm.mockCall(
            lockbox,
            abi.encodeCall(IETHLockbox.authorizedPortals, (IOptimismPortal2(payable(portal2)))),
            abi.encode(false)
        );

        vm.expectRevert(MessageExpiryHub.MessageExpiryHub_UnauthorizedPortal.selector);
        messageExpiryHub.forwardExpiryNotice(lockbox, ATTESTOR_CHAIN_ID, SOURCE_CHAIN_ID, msgHash, MIN_GAS_LIMIT);
    }
}

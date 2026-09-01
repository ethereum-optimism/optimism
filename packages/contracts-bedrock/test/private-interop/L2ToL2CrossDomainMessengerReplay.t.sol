// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { Vm } from "forge-std/Vm.sol";

// Contracts
import { Proxy } from "src/universal/Proxy.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { L2ToL2CrossDomainMessengerReplay } from "src/private-interop/L2ToL2CrossDomainMessengerReplay.sol";

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { IL2ToL2CrossDomainMessengerReplay } from "interfaces/private-interop/IL2ToL2CrossDomainMessengerReplay.sol";

/// @title L2ToL2CrossDomainMessengerReplay_TestInit
/// @notice Reusable test initialization for `L2ToL2CrossDomainMessengerReplay` tests.
abstract contract L2ToL2CrossDomainMessengerReplay_TestInit is Test {
    /// @notice Emitted whenever a message is sent to a destination. Declared here with the stock
    ///         messenger's shape so `vm.expectEmit` checks the replayed log against it.
    event SentMessage(
        uint256 indexed destination, address indexed target, uint256 indexed messageNonce, address sender, bytes message
    );

    /// @notice Replay messenger under test, behind a proxy.
    IL2ToL2CrossDomainMessengerReplay internal replayMessenger;

    /// @notice ProxyAdmin that owns the proxy.
    ProxyAdmin internal proxyAdmin;

    /// @notice Test setup.
    function setUp() public virtual {
        proxyAdmin = new ProxyAdmin(makeAddr("proxyAdminOwner"));
        Proxy proxy = new Proxy(address(proxyAdmin));
        L2ToL2CrossDomainMessengerReplay impl = new L2ToL2CrossDomainMessengerReplay();

        vm.prank(address(proxyAdmin));
        proxy.upgradeTo(address(impl));

        replayMessenger = IL2ToL2CrossDomainMessengerReplay(address(proxy));
    }
}

/// @title L2ToL2CrossDomainMessengerReplay_ReplaySentMessage_Test
/// @notice Tests the `replaySentMessage` function of the `L2ToL2CrossDomainMessengerReplay`
///         contract.
contract L2ToL2CrossDomainMessengerReplay_ReplaySentMessage_Test is L2ToL2CrossDomainMessengerReplay_TestInit {
    /// @notice Tests that the replayer can replay a message and that the returned hash is the
    ///         stock message hash over the public rendering's own chain ID.
    function testFuzz_replaySentMessage_succeeds(
        uint256 _destination,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message
    )
        external
    {
        vm.assume(_sender != Predeploys.SUPERCHAIN_ETH_BRIDGE);
        vm.assume(_target != Predeploys.SUPERCHAIN_ETH_BRIDGE);

        vm.expectEmit(address(replayMessenger));
        emit SentMessage(_destination, _target, _nonce, _sender, _message);

        bytes32 messageHash = replayMessenger.replaySentMessage(_destination, _nonce, _sender, _target, _message);

        assertEq(
            messageHash,
            Hashing.hashL2toL2CrossDomainMessage({
                _destination: _destination,
                _source: block.chainid,
                _nonce: _nonce,
                _sender: _sender,
                _target: _target,
                _message: _message
            })
        );
    }

    /// @notice Tests that a message whose embedded sender is the `SuperchainETHBridge` predeploy is
    ///         refused. The private chain is a custom gas token chain, so the protocol ETH path
    ///         must never be rendered into a public, relayable log.
    function testFuzz_replaySentMessage_ethBridgeSender_reverts(
        uint256 _destination,
        uint256 _nonce,
        address _target,
        bytes calldata _message
    )
        external
    {
        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_ETHBridgeSender.selector);
        replayMessenger.replaySentMessage(_destination, _nonce, Predeploys.SUPERCHAIN_ETH_BRIDGE, _target, _message);
    }

    /// @notice Tests that a message whose embedded target is the `SuperchainETHBridge` predeploy is
    ///         refused, symmetrically with the sender deny. The receiving bridge's own
    ///         `InvalidCrossDomainSender` check makes this redundant in theory; the point of a deny
    ///         list is that it holds without anyone having to reason about what the receiver checks.
    function testFuzz_replaySentMessage_ethBridgeTarget_reverts(
        uint256 _destination,
        uint256 _nonce,
        address _sender,
        bytes calldata _message
    )
        external
    {
        vm.assume(_sender != Predeploys.SUPERCHAIN_ETH_BRIDGE);

        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_ETHBridgeTarget.selector);
        replayMessenger.replaySentMessage(_destination, _nonce, _sender, Predeploys.SUPERCHAIN_ETH_BRIDGE, _message);
    }

    /// @notice Tests that a replayed log is byte-identical to the log the stock messenger emits for
    ///         the same message. Both logs are produced at the standard messenger predeploy address
    ///         — first by the stock implementation, then by the replay implementation installed in
    ///         its place, which is exactly how the public rendering's genesis is arranged — so the
    ///         emitter, every topic and the whole data section are compared directly.
    function test_replaySentMessage_matchesStockMessengerLog_succeeds() external {
        uint256 destination = 420;
        address sender = makeAddr("messageSender");
        address target = makeAddr("messageTarget");
        bytes memory message = hex"deadbeefcafe0123456789";

        // Install the stock messenger at its predeploy address and send a message through it.
        vm.etch(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            vm.getDeployedCode("L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger")
        );
        uint256 nonce = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).messageNonce();

        vm.recordLogs();
        vm.prank(sender);
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).sendMessage(
            destination, target, message
        );
        Vm.Log[] memory stockLogs = vm.getRecordedLogs();
        assertEq(stockLogs.length, 1);
        assertEq(stockLogs[0].topics.length, 4);

        // Install the replay implementation in its place and replay the same message.
        vm.etch(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            vm.getDeployedCode("L2ToL2CrossDomainMessengerReplay.sol:L2ToL2CrossDomainMessengerReplay")
        );
        vm.recordLogs();
        IL2ToL2CrossDomainMessengerReplay(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).replaySentMessage(
            destination, nonce, sender, target, message
        );
        Vm.Log[] memory replayLogs = vm.getRecordedLogs();
        assertEq(replayLogs.length, 1);

        // Byte identity: same emitter, same topics, same data.
        assertEq(replayLogs[0].emitter, stockLogs[0].emitter);
        assertEq(replayLogs[0].topics.length, stockLogs[0].topics.length);
        for (uint256 i = 0; i < stockLogs[0].topics.length; i++) {
            assertEq(replayLogs[0].topics[i], stockLogs[0].topics[i]);
        }
        assertEq(replayLogs[0].data, stockLogs[0].data);
    }
}

/// @title L2ToL2CrossDomainMessengerReplay_Uncategorized_Test
/// @notice Tests the stock messenger entry points that the `L2ToL2CrossDomainMessengerReplay`
///         contract deliberately does not support.
contract L2ToL2CrossDomainMessengerReplay_Uncategorized_Test is L2ToL2CrossDomainMessengerReplay_TestInit {
    /// @notice Tests that the version is set.
    function test_version_succeeds() external view {
        assertTrue(bytes(replayMessenger.version()).length > 0);
    }

    /// @notice Tests that the message version matches the stock messenger's.
    function test_messageVersion_succeeds() external view {
        assertEq(replayMessenger.messageVersion(), uint16(0));
    }

    /// @notice Tests that `sendMessage` is unsupported.
    function test_sendMessage_unsupported_reverts() external {
        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_Unsupported.selector);
        replayMessenger.sendMessage(420, address(0xbeef), hex"1234");
    }

    /// @notice Tests that `resendMessage` is unsupported.
    function test_resendMessage_unsupported_reverts() external {
        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_Unsupported.selector);
        replayMessenger.resendMessage(420, 0, address(0xbeef), address(0xcafe), hex"1234");
    }

    /// @notice Tests that `relayMessage` is unsupported.
    function test_relayMessage_unsupported_reverts() external {
        Identifier memory id = Identifier({
            origin: Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            blockNumber: 1,
            logIndex: 0,
            timestamp: 1,
            chainId: 420
        });

        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_Unsupported.selector);
        replayMessenger.relayMessage(id, hex"1234");
    }

    /// @notice Tests that the cross domain message context getters are unsupported.
    function test_crossDomainMessageContext_unsupported_reverts() external {
        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_Unsupported.selector);
        replayMessenger.crossDomainMessageSender();

        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_Unsupported.selector);
        replayMessenger.crossDomainMessageSource();

        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_Unsupported.selector);
        replayMessenger.crossDomainMessageContext();
    }

    /// @notice Tests that the message accounting getters revert rather than reporting a zero value
    ///         that a caller could mistake for a real answer.
    function test_messageNonce_unsupported_reverts() external {
        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_Unsupported.selector);
        replayMessenger.messageNonce();

        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_Unsupported.selector);
        replayMessenger.successfulMessages(bytes32(0));

        vm.expectRevert(IL2ToL2CrossDomainMessengerReplay.L2ToL2CrossDomainMessengerReplay_Unsupported.selector);
        replayMessenger.sentMessages(0);
    }
}

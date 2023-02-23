// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import { Messenger_Initializer } from "./CommonTest.t.sol";
import { L1ERC721Bridge } from "../L1/L1ERC721Bridge.sol";
import { L2ERC721Bridge } from "../L2/L2ERC721Bridge.sol";

contract TestERC721 is ERC721 {
    constructor() ERC721("Test", "TST") {}

    function mint(address to, uint256 tokenId) public {
        _mint(to, tokenId);
    }
}

contract L1ERC721Bridge_Test is Messenger_Initializer {
    TestERC721 internal localToken;
    TestERC721 internal remoteToken;
    L1ERC721Bridge internal bridge;
    address internal constant otherBridge = address(0x3456);
    uint256 internal constant tokenId = 1;

    event ERC721BridgeInitiated(
        address indexed localToken,
        address indexed remoteToken,
        address indexed from,
        address to,
        uint256 tokenId,
        bytes extraData
    );

    event ERC721BridgeFinalized(
        address indexed localToken,
        address indexed remoteToken,
        address indexed from,
        address to,
        uint256 tokenId,
        bytes extraData
    );

    function setUp() public override {
        super.setUp();

        // Create necessary contracts.
        bridge = new L1ERC721Bridge(address(L1Messenger), otherBridge);
        localToken = new TestERC721();
        remoteToken = new TestERC721();

        // Label the bridge so we get nice traces.
        vm.label(address(bridge), "L1ERC721Bridge");

        // Mint alice a token.
        localToken.mint(alice, tokenId);

        // Approve the bridge to transfer the token.
        vm.prank(alice);
        localToken.approve(address(bridge), tokenId);
    }

    function test_constructor_succeeds() public {
        assertEq(address(bridge.MESSENGER()), address(L1Messenger));
        assertEq(address(bridge.OTHER_BRIDGE()), otherBridge);
        assertEq(address(bridge.messenger()), address(L1Messenger));
        assertEq(address(bridge.otherBridge()), otherBridge);
    }

    function test_bridgeERC721_succeeds() public {
        // Expect a call to the messenger.
        vm.expectCall(
            address(L1Messenger),
            abi.encodeCall(
                L1Messenger.sendMessage,
                (
                    address(otherBridge),
                    abi.encodeCall(
                        L2ERC721Bridge.finalizeBridgeERC721,
                        (
                            address(remoteToken),
                            address(localToken),
                            alice,
                            alice,
                            tokenId,
                            hex"5678"
                        )
                    ),
                    1234
                )
            )
        );

        // Expect an event to be emitted.
        vm.expectEmit(true, true, true, true);
        emit ERC721BridgeInitiated(
            address(localToken),
            address(remoteToken),
            alice,
            alice,
            tokenId,
            hex"5678"
        );

        // Bridge the token.
        vm.prank(alice);
        bridge.bridgeERC721(address(localToken), address(remoteToken), tokenId, 1234, hex"5678");

        // Token is locked in the bridge.
        assertEq(bridge.deposits(address(localToken), address(remoteToken), tokenId), true);
        assertEq(localToken.ownerOf(tokenId), address(bridge));
    }

    function test_bridgeERC721_fromContract_reverts() external {
        // Bridge the token.
        vm.etch(alice, hex"01");
        vm.prank(alice);
        vm.expectRevert("ERC721Bridge: account is not externally owned");
        bridge.bridgeERC721(address(localToken), address(remoteToken), tokenId, 1234, hex"5678");

        // Token is not locked in the bridge.
        assertEq(bridge.deposits(address(localToken), address(remoteToken), tokenId), false);
        assertEq(localToken.ownerOf(tokenId), alice);
    }

    function test_bridgeERC721_localTokenZeroAddress_reverts() external {
        // Bridge the token.
        vm.prank(alice);
        vm.expectRevert();
        bridge.bridgeERC721(address(0), address(remoteToken), tokenId, 1234, hex"5678");

        // Token is not locked in the bridge.
        assertEq(bridge.deposits(address(localToken), address(remoteToken), tokenId), false);
        assertEq(localToken.ownerOf(tokenId), alice);
    }

    function test_bridgeERC721_remoteTokenZeroAddress_reverts() external {
        // Bridge the token.
        vm.prank(alice);
        vm.expectRevert("L1ERC721Bridge: remote token cannot be address(0)");
        bridge.bridgeERC721(address(localToken), address(0), tokenId, 1234, hex"5678");

        // Token is not locked in the bridge.
        assertEq(bridge.deposits(address(localToken), address(remoteToken), tokenId), false);
        assertEq(localToken.ownerOf(tokenId), alice);
    }

    function test_bridgeERC721_wrongOwner_reverts() external {
        // Bridge the token.
        vm.prank(bob);
        vm.expectRevert("ERC721: transfer from incorrect owner");
        bridge.bridgeERC721(address(localToken), address(remoteToken), tokenId, 1234, hex"5678");

        // Token is not locked in the bridge.
        assertEq(bridge.deposits(address(localToken), address(remoteToken), tokenId), false);
        assertEq(localToken.ownerOf(tokenId), alice);
    }

    function test_bridgeERC721To_succeeds() external {
        // Expect a call to the messenger.
        vm.expectCall(
            address(L1Messenger),
            abi.encodeCall(
                L1Messenger.sendMessage,
                (
                    address(otherBridge),
                    abi.encodeCall(
                        L2ERC721Bridge.finalizeBridgeERC721,
                        (address(remoteToken), address(localToken), alice, bob, tokenId, hex"5678")
                    ),
                    1234
                )
            )
        );

        // Expect an event to be emitted.
        vm.expectEmit(true, true, true, true);
        emit ERC721BridgeInitiated(
            address(localToken),
            address(remoteToken),
            alice,
            bob,
            tokenId,
            hex"5678"
        );

        // Bridge the token.
        vm.prank(alice);
        bridge.bridgeERC721To(
            address(localToken),
            address(remoteToken),
            bob,
            tokenId,
            1234,
            hex"5678"
        );

        // Token is locked in the bridge.
        assertEq(bridge.deposits(address(localToken), address(remoteToken), tokenId), true);
        assertEq(localToken.ownerOf(tokenId), address(bridge));
    }

    function test_bridgeERC721To_localTokenZeroAddress_reverts() external {
        // Bridge the token.
        vm.prank(alice);
        vm.expectRevert();
        bridge.bridgeERC721To(address(0), address(remoteToken), bob, tokenId, 1234, hex"5678");

        // Token is not locked in the bridge.
        assertEq(bridge.deposits(address(localToken), address(remoteToken), tokenId), false);
        assertEq(localToken.ownerOf(tokenId), alice);
    }

    function test_bridgeERC721To_remoteTokenZeroAddress_reverts() external {
        // Bridge the token.
        vm.prank(alice);
        vm.expectRevert("L1ERC721Bridge: remote token cannot be address(0)");
        bridge.bridgeERC721To(address(localToken), address(0), bob, tokenId, 1234, hex"5678");

        // Token is not locked in the bridge.
        assertEq(bridge.deposits(address(localToken), address(remoteToken), tokenId), false);
        assertEq(localToken.ownerOf(tokenId), alice);
    }

    function test_bridgeERC721To_wrongOwner_reverts() external {
        // Bridge the token.
        vm.prank(bob);
        vm.expectRevert("ERC721: transfer from incorrect owner");
        bridge.bridgeERC721To(
            address(localToken),
            address(remoteToken),
            bob,
            tokenId,
            1234,
            hex"5678"
        );

        // Token is not locked in the bridge.
        assertEq(bridge.deposits(address(localToken), address(remoteToken), tokenId), false);
        assertEq(localToken.ownerOf(tokenId), alice);
    }

    function test_finalizeBridgeERC721_succeeds() external {
        // Bridge the token.
        vm.prank(alice);
        bridge.bridgeERC721(address(localToken), address(remoteToken), tokenId, 1234, hex"5678");

        // Expect an event to be emitted.
        vm.expectEmit(true, true, true, true);
        emit ERC721BridgeFinalized(
            address(localToken),
            address(remoteToken),
            alice,
            alice,
            tokenId,
            hex"5678"
        );

        // Finalize a withdrawal.
        vm.mockCall(
            address(L1Messenger),
            abi.encodeWithSelector(L1Messenger.xDomainMessageSender.selector),
            abi.encode(otherBridge)
        );
        vm.prank(address(L1Messenger));
        bridge.finalizeBridgeERC721(
            address(localToken),
            address(remoteToken),
            alice,
            alice,
            tokenId,
            hex"5678"
        );

        // Token is not locked in the bridge.
        assertEq(bridge.deposits(address(localToken), address(remoteToken), tokenId), false);
        assertEq(localToken.ownerOf(tokenId), alice);
    }

    function test_finalizeBridgeERC721_notViaLocalMessenger_reverts() external {
        // Finalize a withdrawal.
        vm.prank(alice);
        vm.expectRevert("ERC721Bridge: function can only be called from the other bridge");
        bridge.finalizeBridgeERC721(
            address(localToken),
            address(remoteToken),
            alice,
            alice,
            tokenId,
            hex"5678"
        );
    }

    function test_finalizeBridgeERC721_notFromRemoteMessenger_reverts() external {
        // Finalize a withdrawal.
        vm.mockCall(
            address(L1Messenger),
            abi.encodeWithSelector(L1Messenger.xDomainMessageSender.selector),
            abi.encode(alice)
        );
        vm.prank(address(L1Messenger));
        vm.expectRevert("ERC721Bridge: function can only be called from the other bridge");
        bridge.finalizeBridgeERC721(
            address(localToken),
            address(remoteToken),
            alice,
            alice,
            tokenId,
            hex"5678"
        );
    }

    function test_finalizeBridgeERC721_selfToken_reverts() external {
        // Finalize a withdrawal.
        vm.mockCall(
            address(L1Messenger),
            abi.encodeWithSelector(L1Messenger.xDomainMessageSender.selector),
            abi.encode(otherBridge)
        );
        vm.prank(address(L1Messenger));
        vm.expectRevert("L1ERC721Bridge: local token cannot be self");
        bridge.finalizeBridgeERC721(
            address(bridge),
            address(remoteToken),
            alice,
            alice,
            tokenId,
            hex"5678"
        );
    }

    function test_finalizeBridgeERC721_notEscrowed_reverts() external {
        // Finalize a withdrawal.
        vm.mockCall(
            address(L1Messenger),
            abi.encodeWithSelector(L1Messenger.xDomainMessageSender.selector),
            abi.encode(otherBridge)
        );
        vm.prank(address(L1Messenger));
        vm.expectRevert("L1ERC721Bridge: Token ID is not escrowed in the L1 Bridge");
        bridge.finalizeBridgeERC721(
            address(localToken),
            address(remoteToken),
            alice,
            alice,
            tokenId,
            hex"5678"
        );
    }
}

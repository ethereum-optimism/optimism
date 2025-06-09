// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Contracts
import { SuperchainNFTBridge } from "src/L2/SuperchainNFTBridge.sol";
import { MockSuperchainERC721Implementation } from "test/mocks/SuperchainERC721Implementation.sol";
import { ERC721 } from "@solady-v0.0.245/tokens/ERC721.sol";

// Interfaces
import { ISuperchainNFTBridge } from "interfaces/L2/ISuperchainNFTBridge.sol";
import { ICrosschainERC721 } from "interfaces/L2/ICrosschainERC721.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { ISuperchainERC721 } from "interfaces/L2/ISuperchainERC721.sol";

contract SuperchainNFTBridge_TestInit is CommonTest {
    address internal constant ZERO_ADDRESS = address(0);

    // Mocks
    ISuperchainERC721 public superchainERC721;

    // Events
    event SendERC721(
        address indexed token, address indexed from, address indexed to, uint256 tokenId, uint256 destination
    );
    event RelayERC721(address indexed token, address indexed from, address indexed to, uint256 tokenId, uint256 source);

    function setUp() public virtual override {
        useInteropOverride = true;
        super.setUp();

        vm.etch(Predeploys.SUPERCHAIN_NFT_BRIDGE, address(new SuperchainNFTBridge()).code);
        superchainERC721 = ISuperchainERC721(address(new MockSuperchainERC721Implementation()));
    }

    /// @notice Helper function to setup a mock and expect a call to it.
    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }
}

contract SuperchainNFTBridge_SendERC721_Test is SuperchainNFTBridge_TestInit {
    /// @notice Tests the `sendERC721` function reverts when the address `_to` is zero.
    function testFuzz_sendERC721_toIsZeroAddress_reverts(address _sender, uint256 _tokenId, uint256 _chainId) public {
        // Expect the revert with `ZeroAddress` selector
        vm.expectRevert(ISuperchainNFTBridge.ZeroAddress.selector);

        // Call the `sendERC721` function
        vm.prank(_sender);
        superchainNFTBridge.sendERC721(address(superchainERC721), ZERO_ADDRESS, _tokenId, _chainId);
    }

    /// @notice Tests the `sendERC721` function reverts when the `_nft` does not support the
    ///         `ICrosschainERC721` interface.
    function testFuzz_sendERC721_notSupportedCrosschainERC721_reverts(
        address _nft,
        address _sender,
        address _to,
        uint256 _tokenId,
        uint256 _chainId
    )
        public
    {
        vm.assume(_to != ZERO_ADDRESS);
        assumeAddressIsNot(_nft, AddressType.Precompile, AddressType.ForgeAddress);

        // Mock the call over the `supportsInterface` function to return false
        _mockAndExpect(
            _nft,
            abi.encodeCall(ISuperchainERC721.supportsInterface, (type(ICrosschainERC721).interfaceId)),
            abi.encode(false)
        );

        // Expect the revert with `InvalidCrosschainERC721` selector
        vm.expectRevert(ISuperchainNFTBridge.InvalidCrosschainERC721.selector);

        // Call the `sendERC721` function
        vm.prank(_sender);
        superchainNFTBridge.sendERC721(_nft, _to, _tokenId, _chainId);
    }

    /// @notice Tests the `sendERC721` function burns the sender token, sends the message, and
    ///         emits the `SendERC721` event.
    function test_sendERC721_succeeds(
        address _sender,
        address _to,
        uint256 _tokenId,
        uint256 _chainId,
        bytes32 _msgHash
    )
        public
    {
        vm.assume(_sender != ZERO_ADDRESS);
        vm.assume(_to != ZERO_ADDRESS);

        // Make `_sender` owner of `_tokenId`
        vm.prank(address(superchainNFTBridge));
        superchainERC721.crosschainMint(_sender, _tokenId);

        assertEq(superchainERC721.ownerOf(_tokenId), _sender);

        // Mock the call over the `sendMessage` function
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(
                IL2ToL2CrossDomainMessenger.sendMessage,
                (
                    _chainId,
                    address(superchainNFTBridge),
                    abi.encodeCall(
                        ISuperchainNFTBridge.relayERC721, (address(superchainERC721), _sender, _to, _tokenId)
                    )
                )
            ),
            abi.encode(_msgHash)
        );

        // Expect the `SendERC721` event
        vm.expectEmit(address(superchainNFTBridge));
        emit SendERC721(address(superchainERC721), _sender, _to, _tokenId, _chainId);

        // Call the `sendERC721` function
        vm.prank(_sender);
        bytes32 _returnedMsgHash = superchainNFTBridge.sendERC721(address(superchainERC721), _to, _tokenId, _chainId);

        // Check the message hash was generated correctly
        assertEq(_msgHash, _returnedMsgHash);

        // Check the token was sent and is burnt
        vm.expectRevert(ERC721.TokenDoesNotExist.selector);
        superchainERC721.ownerOf(_tokenId);
    }
}

contract SuperchainNFTBridge_RelayERC721_Test is SuperchainNFTBridge_TestInit {
    /// @notice Tests the `relayERC721` function reverts when the `_sender` is not the messenger.
    function testFuzz_relayERC721_notMessenger_reverts(
        address _sender,
        address _from,
        address _to,
        uint256 _tokenId
    )
        public
    {
        vm.assume(_sender != ZERO_ADDRESS);
        vm.assume(_sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        // Expect the revert with `Unauthorized` selector
        vm.expectRevert(ISuperchainNFTBridge.Unauthorized.selector);

        // Call the `relayERC721` function
        vm.prank(_sender);
        superchainNFTBridge.relayERC721(address(superchainERC721), _from, _to, _tokenId);
    }

    /// @notice Tests the `relayERC721` function reverts when the `_crossDomainMessageSender` is not
    ///         the bridge.
    function testFuzz_relayERC721_notCrossDomainMessageSender_reverts(
        address _from,
        address _to,
        uint256 _tokenId,
        address _crossDomainMessageSender,
        uint256 _source
    )
        public
    {
        vm.assume(_crossDomainMessageSender != address(superchainNFTBridge));

        // Mock the call over the `crossDomainMessageContext` function setting a wrong sender
        _mockAndExpect(
            address(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER),
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(_crossDomainMessageSender, _source)
        );

        // Expect the revert with `InvalidCrossDomainSender` selector
        vm.expectRevert(ISuperchainNFTBridge.InvalidCrossDomainSender.selector);

        // Call the `relayERC721` function
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        superchainNFTBridge.relayERC721(address(superchainERC721), _from, _to, _tokenId);
    }

    /// @notice Tests the `relayERC20` mints the proper amount and emits the `RelayERC20` event.
    function testFuzz_relayERC721_succeeds(address _from, address _to, uint256 _tokenId, uint256 _source) public {
        vm.assume(_to != ZERO_ADDRESS);

        // Mock the call over the `crossDomainMessageContext` function setting the same address as
        // value
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(address(superchainNFTBridge), _source)
        );

        // Look for the emit of the `RelayERC721` event
        vm.expectEmit(address(superchainNFTBridge));
        emit RelayERC721(address(superchainERC721), _from, _to, _tokenId, _source);

        // Call the `relayERC20` function with the messenger caller
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        superchainNFTBridge.relayERC721(address(superchainERC721), _from, _to, _tokenId);

        // Check the token was minted to `_to`
        assertEq(superchainERC721.ownerOf(_tokenId), _to);
    }
}

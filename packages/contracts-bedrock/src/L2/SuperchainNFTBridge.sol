// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { ZeroAddress, Unauthorized } from "src/libraries/errors/CommonErrors.sol";

// Interfaces
import { ICrosschainERC721, IERC165 } from "interfaces/L2/ICrosschainERC721.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000029
/// @title SuperchainNFTBridge
/// @notice The SuperchainNFTBridge allows for the bridging of ERC721 tokens across the Superchain.
///         It builds on top of the L2ToL2CrossDomainMessenger for both replay protection and domain
///         binding.
contract SuperchainNFTBridge {
    /// @notice Thrown when attempting to relay a message and the cross domain message sender is not the
    /// SuperchainNFTBridge.
    error InvalidCrossDomainSender();

    /// @notice Thrown when attempting to use a token that does not implement the ICrosschainERC721 interface.
    error InvalidCrosschainERC721();

    /// @notice Emitted when an NFT is sent from one chain to another.
    /// @param token         Address of the token sent.
    /// @param from          Address of the sender.
    /// @param to            Address of the recipient.
    /// @param tokenId       Token ID being sent.
    /// @param destination   Chain ID of the destination chain.
    event SendERC721(
        address indexed token, address indexed from, address indexed to, uint256 tokenId, uint256 destination
    );

    /// @notice Emitted whenever an NFT is successfully relayed on this chain.
    /// @param token         Address of the token relayed.
    /// @param from          Address of the msg.sender of sendERC721 on the source chain.
    /// @param to            Address of the recipient.
    /// @param tokenId       Token ID being relayed.
    /// @param source        Chain ID of the source chain.
    event RelayERC721(address indexed token, address indexed from, address indexed to, uint256 tokenId, uint256 source);

    /// @notice Address of the L2ToL2CrossDomainMessenger Predeploy.
    address internal constant MESSENGER = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;

    /// @notice Semantic version.
    /// @custom:semver 0.0.1
    string public constant version = "0.0.1";

    /// @notice Sends an NFT to a target address on another chain.
    /// @dev The NFT is burned on the source chain.
    /// @param _token    Token to send.
    /// @param _to       Address to send the NFT to.
    /// @param _tokenId  Token ID being sent.
    /// @param _chainId  Chain ID of the destination chain.
    /// @return msgHash_ Hash of the message sent.
    function sendERC721(
        address _token,
        address _to,
        uint256 _tokenId,
        uint256 _chainId
    )
        external
        returns (bytes32 msgHash_)
    {
        if (_to == address(0)) revert ZeroAddress();

        if (!IERC165(_token).supportsInterface(type(ICrosschainERC721).interfaceId)) revert InvalidCrosschainERC721();

        ICrosschainERC721(_token).crosschainBurn(msg.sender, _tokenId);

        bytes memory message = abi.encodeCall(this.relayERC721, (_token, msg.sender, _to, _tokenId));
        msgHash_ = IL2ToL2CrossDomainMessenger(MESSENGER).sendMessage(_chainId, address(this), message);

        emit SendERC721(_token, msg.sender, _to, _tokenId, _chainId);
    }

    /// @notice Relays an NFT received from another chain.
    /// @dev The NFT is minted on the destination chain.
    /// @param _token   Token to relay.
    /// @param _from    Address of the msg.sender of sendERC721 on the source chain.
    /// @param _to      Address to relay the NFT to.
    /// @param _tokenId Token ID being relayed.
    function relayERC721(address _token, address _from, address _to, uint256 _tokenId) external {
        if (msg.sender != MESSENGER) revert Unauthorized();

        (address crossDomainMessageSender, uint256 source) =
            IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageContext();

        if (crossDomainMessageSender != address(this)) revert InvalidCrossDomainSender();

        ICrosschainERC721(_token).crosschainMint(_to, _tokenId);

        emit RelayERC721(_token, _from, _to, _tokenId, source);
    }
}

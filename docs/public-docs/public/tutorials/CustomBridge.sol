// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Libraries
import { PredeployAddresses } from "interop-lib/src/libraries/PredeployAddresses.sol";

// Interfaces
import { IERC7802, IERC165 } from "interop-lib/src/interfaces/IERC7802.sol";
import { IL2ToL2CrossDomainMessenger } from "interop-lib/src/interfaces/IL2ToL2CrossDomainMessenger.sol";

/// @custom:proxied true
/// @title CustomBridge
contract CustomBridge {
    // Immutable configuration
    address public immutable tokenAddressHere;
    address public immutable tokenAddressThere;
    uint256 public immutable chainIdThere;
    address public immutable bridgeAddressThere;

    error ZeroAddress();
    error Unauthorized();

    /// @notice Thrown when attempting to relay a message and the cross domain message sender is not the
    /// SuperchainTokenBridge.
    error InvalidCrossDomainSender();

    /// @notice Emitted when tokens are sent from one chain to another.
    /// @param token         Address of the token sent.
    /// @param from          Address of the sender.
    /// @param to            Address of the recipient.
    /// @param amount        Number of tokens sent.
    /// @param destination   Chain ID of the destination chain.
    event SendERC20(
        address indexed token, address indexed from, address indexed to, uint256 amount, uint256 destination
    );

    /// @notice Emitted whenever tokens are successfully relayed on this chain.
    /// @param token         Address of the token relayed.
    /// @param from          Address of the msg.sender of sendERC20 on the source chain.
    /// @param to            Address of the recipient.
    /// @param amount        Amount of tokens relayed.
    /// @param source        Chain ID of the source chain.
    event RelayERC20(address indexed token, address indexed from, address indexed to, uint256 amount, uint256 source);

    /// @notice Address of the L2ToL2CrossDomainMessenger Predeploy.
    address internal constant MESSENGER = PredeployAddresses.L2_TO_L2_CROSS_DOMAIN_MESSENGER;

    // Setup the configuration
    constructor(
        address tokenAddressHere_,
        address tokenAddressThere_,
        uint256 chainIdThere_,
        address bridgeAddressThere_        
    ) {
        if (
            tokenAddressHere_ == address(0) ||
            tokenAddressThere_ == address(0) ||
            bridgeAddressThere_ == address(0)
        ) revert ZeroAddress();

        tokenAddressHere = tokenAddressHere_;
        tokenAddressThere = tokenAddressThere_;
        chainIdThere = chainIdThere_;
        bridgeAddressThere = bridgeAddressThere_;
    }    

    /// @notice Sends tokens to a target address on another chain.
    /// @dev Tokens are burned on the source chain.
    /// @param _to       Address to send tokens to.
    /// @param _amount   Amount of tokens to send.
    /// @return msgHash_ Hash of the message sent.
    function sendERC20(
        address _to,
        uint256 _amount
    )
        external
        returns (bytes32 msgHash_)
    {
        if (_to == address(0)) revert ZeroAddress();

        IERC7802(tokenAddressHere).crosschainBurn(msg.sender, _amount);

        bytes memory message = abi.encodeCall(this.relayERC20, (msg.sender, _to, _amount));
        msgHash_ = IL2ToL2CrossDomainMessenger(MESSENGER).sendMessage(chainIdThere, bridgeAddressThere, message);

        emit SendERC20(tokenAddressHere, msg.sender, _to, _amount, chainIdThere);
    }

    /// @notice Relays tokens received from another chain.
    /// @dev Tokens are minted on the destination chain.
    /// @param _from    Address of the msg.sender of sendERC20 on the source chain.
    /// @param _to      Address to relay tokens to.
    /// @param _amount  Amount of tokens to relay.
    function relayERC20(address _from, address _to, uint256 _amount) external {
        if (msg.sender != MESSENGER) revert Unauthorized();

        (address crossDomainMessageSender, uint256 source) =
            IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageContext();

        if (crossDomainMessageSender != bridgeAddressThere) revert InvalidCrossDomainSender();
        if (source != chainIdThere) revert InvalidCrossDomainSender();

        IERC7802(tokenAddressHere).crosschainMint(_to, _amount);

        emit RelayERC20(tokenAddressHere, _from, _to, _amount, chainIdThere);
    }
}
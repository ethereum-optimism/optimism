// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CrossDomainMessenger } from "./CrossDomainMessenger.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";

/// @title ERC1155Bridge
/// @notice ERC1155Bridge is a base contract for the L1 and L2 ERC1155 bridges.
abstract contract ERC1155Bridge {
    /// @notice Messenger contract on this domain.
    CrossDomainMessenger public immutable MESSENGER;

    /// @notice Address of the bridge on the other network.
    address public immutable OTHER_BRIDGE;

    /// @notice Reserve extra slots (to a total of 50) in the storage layout for future upgrades.
    uint256[50] private __gap;

    /// @notice Emitted when an ERC1155 bridge to the other network is initiated.
    /// @param localToken  Address of the token on this domain.
    /// @param remoteToken Address of the token on the remote domain.
    /// @param from        Address that initiated bridging action.
    /// @param to          Address to receive the token.
    /// @param id          Type ID of the token deposited.
    /// @param value       Amount of tokens deposited.
    /// @param extraData   Extra data for use on the client-side.
    event ERC1155BridgeInitiated(
        address indexed localToken,
        address indexed remoteToken,
        address indexed from,
        address to,
        uint256 id,
        uint256 value,
        bytes extraData
    );

    /// @notice Emitted when an ERC1155 bridge from the other network is finalized.
    /// @param localToken  Address of the token on this domain.
    /// @param remoteToken Address of the token on the remote domain.
    /// @param from        Address that initiated bridging action.
    /// @param to          Address to receive the token.
    /// @param id          Type ID of the token deposited.
    /// @param value       Amount of tokens deposited.
    /// @param extraData   Extra data for use on the client-side.
    event ERC1155BridgeFinalized(
        address indexed localToken,
        address indexed remoteToken,
        address indexed from,
        address to,
        uint256 id,
        uint256 value,
        bytes extraData
    );

    /// @notice Ensures that the caller is a cross-chain message from the other bridge.
    modifier onlyOtherBridge() {
        require(
            msg.sender == address(MESSENGER) && MESSENGER.xDomainMessageSender() == OTHER_BRIDGE,
            "ERC1155Bridge: function can only be called from the other bridge"
        );
        _;
    }

    /// @param _messenger   Address of the CrossDomainMessenger on this network.
    /// @param _otherBridge Address of the ERC1155 bridge on the other network.
    constructor(address _messenger, address _otherBridge) {
        require(_messenger != address(0), "ERC1155Bridge: messenger cannot be address(0)");
        require(_otherBridge != address(0), "ERC1155Bridge: other bridge cannot be address(0)");

        MESSENGER = CrossDomainMessenger(_messenger);
        OTHER_BRIDGE = _otherBridge;
    }

    /// @custom:legacy
    /// @notice Legacy getter for messenger contract.
    /// @return Messenger contract on this domain.
    function messenger() external view returns (CrossDomainMessenger) {
        return MESSENGER;
    }

    /// @custom:legacy
    /// @notice Legacy getter for other bridge address.
    /// @return Address of the bridge on the other network.
    function otherBridge() external view returns (address) {
        return OTHER_BRIDGE;
    }

    /// @notice Initiates a bridge of an ERC1155 to the caller's account on the other chain. Note
    ///         that this function can only be called by EOAs. Smart contract wallets should use the
    ///         `bridgeERC1155To` function after ensuring that the recipient address on the remote
    ///         chain exists. Also note that the current owner of the tokens on this chain must
    ///         approve this contract to operate the tokens before it can be bridged.
    ///         **WARNING**: Do not bridge an ERC1155 that was originally deployed on Optimism. This
    ///         bridge only supports ERC1155s originally deployed on Ethereum. Users will need to
    ///         wait for the one-week challenge period to elapse before their Optimism-native
    ///         ERC1155 can be refunded on L2.
    /// @param _localToken  Address of the ERC1155 on this domain.
    /// @param _remoteToken Address of the ERC1155 on the remote domain.
    /// @param _id          Type ID of the token to bridge.
    /// @param _amount      Amount of tokens to bridge.
    /// @param _minGasLimit Minimum gas limit for the bridge message on the other domain.
    /// @param _extraData   Optional data to forward to the other chain. Data supplied here will not
    ///                     be used to execute any code on the other chain and is only emitted as
    ///                     extra data for the convenience of off-chain tooling.
    function bridgeERC1155(
        address _localToken,
        address _remoteToken,
        uint256 _id,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) external {
        // Modifier requiring sender to be EOA. This prevents against a user error that would occur
        // if the sender is a smart contract wallet that has a different address on the remote chain
        // (or doesn't have an address on the remote chain at all). The user would fail to receive
        // the tokens if they use this function because it sends the tokens to the same address as
        // the caller. This check could be bypassed by a malicious contract via initcode, but it
        // takes care of the user error we want to avoid.
        require(!Address.isContract(msg.sender), "ERC1155Bridge: account is not externally owned");

        _initiateBridgeERC1155(
            _localToken,
            _remoteToken,
            msg.sender,
            msg.sender,
            _id,
            _amount,
            _minGasLimit,
            _extraData
        );
    }

    /// @notice Initiates a bridge of an ERC1155 to some recipient's account on the other chain.
    ///         Note that the current owner of the tokens on this chain must approve this contract
    ///         to operate the tokens before it can be bridged.
    ///         **WARNING**: Do not bridge an ERC1155 that was originally deployed on Optimism. This
    ///         bridge only supports ERC1155s originally deployed on Ethereum. Users will need to
    ///         wait for the one-week challenge period to elapse before their Optimism-native tokens
    ///         can be refunded on L2.
    /// @param _localToken  Address of the ERC1155 on this domain.
    /// @param _remoteToken Address of the ERC1155 on the remote domain.
    /// @param _to          Address to receive the token on the other domain.
    /// @param _id          Type ID of the token to bridge.
    /// @param _amount      Amount of tokens to bridge.
    /// @param _minGasLimit Minimum gas limit for the bridge message on the other domain.
    /// @param _extraData   Optional data to forward to the other chain. Data supplied here will not
    ///                     be used to execute any code on the other chain and is only emitted as
    ///                     extra data for the convenience of off-chain tooling.
    function bridgeERC1155To(
        address _localToken,
        address _remoteToken,
        address _to,
        uint256 _id,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) external {
        require(_to != address(0), "ERC1155Bridge: nft recipient cannot be address(0)");

        _initiateBridgeERC1155(
            _localToken,
            _remoteToken,
            msg.sender,
            _to,
            _id,
            _amount,
            _minGasLimit,
            _extraData
        );
    }

    /// @notice Internal function for initiating a token bridge to the other domain.
    /// @param _localToken  Address of the ERC1155 on this domain.
    /// @param _remoteToken Address of the ERC1155 on the remote domain.
    /// @param _from        Address of the sender on this domain.
    /// @param _to          Address to receive the token on the other domain.
    /// @param _id          Type ID of the token to bridge.
    /// @param _amount      Amount of tokens to bridge.
    /// @param _minGasLimit Minimum gas limit for the bridge message on the other domain.
    /// @param _extraData   Optional data to forward to the other domain. Data supplied here will
    ///                     not be used to execute any code on the other domain and is only emitted
    ///                     as extra data for the convenience of off-chain tooling.
    function _initiateBridgeERC1155(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _id,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) internal virtual;
}

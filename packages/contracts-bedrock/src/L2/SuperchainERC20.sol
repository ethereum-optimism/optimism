// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { ISuperchainERC20Extensions } from "src/L2/ISuperchainERC20.sol";
import { ERC20 } from "@solady/tokens/ERC20.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/IL2ToL2CrossDomainMessenger.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @notice Thrown when attempting to relay a message and the function caller (msg.sender) is not
/// L2ToL2CrossDomainMessenger.
error CallerNotL2ToL2CrossDomainMessenger();

/// @notice Thrown when attempting to relay a message and the cross domain message sender is not this SuperchainERC20.
error InvalidCrossDomainSender();

/// @notice Thrown when attempting to mint or burn tokens and the account is the zero address.
error ZeroAddress();

/// @title SuperchainERC20
/// @notice SuperchainERC20 is a standard extension of the base ERC20 token contract that unifies ERC20 token
///         bridging to make it fungible across the Superchain. It builds on top of the L2ToL2CrossDomainMessenger for
///         both replay protection and domain binding.
contract SuperchainERC20 is ISuperchainERC20Extensions, ERC20 {
    /// @notice Address of the L2ToL2CrossDomainMessenger Predeploy.
    address internal constant MESSENGER = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;

    /// @notice Storage slot that the SuperchainERC20Metadata struct is stored at.
    /// keccak256(abi.encode(uint256(keccak256("superchainERC20.metadata")) - 1)) & ~bytes32(uint256(0xff));
    bytes32 internal constant SUPERCHAIN_ERC20_METADATA_SLOT =
        0xd17d6ca6a839692cc315581e57453e7dbbeba09485cfb8c48daa1d1181778600;

    /// @notice Storage struct for the SuperchainERC20 metadata.
    /// @custom:storage-location erc7201:superchainERC20.metadata
    struct SuperchainERC20Metadata {
        /// @notice Name of the token
        string name;
        /// @notice Symbol of the token
        string symbol;
        /// @notice Decimals of the token
        uint8 decimals;
    }

    /// @notice Returns the storage for the SuperchainERC20Metadata.
    function _getMetadataStorage() private pure returns (SuperchainERC20Metadata storage _storage) {
        assembly {
            _storage.slot := SUPERCHAIN_ERC20_METADATA_SLOT
        }
    }

    /// @notice Sets the storage for the SuperchainERC20Metadata.
    /// @param _name     Name of the token.
    /// @param _symbol   Symbol of the token.
    /// @param _decimals Decimals of the token.
    function _setMetadataStorage(string memory _name, string memory _symbol, uint8 _decimals) internal {
        SuperchainERC20Metadata storage _storage = _getMetadataStorage();
        _storage.name = _name;
        _storage.symbol = _symbol;
        _storage.decimals = _decimals;
    }

    /// @notice Constructs the SuperchainERC20 contract.
    /// @param _name           ERC20 name.
    /// @param _symbol         ERC20 symbol.
    /// @param _decimals       ERC20 decimals.
    constructor(string memory _name, string memory _symbol, uint8 _decimals) {
        _setMetadataStorage(_name, _symbol, _decimals);
    }

    /// @notice Sends tokens to some target address on another chain.
    /// @param _to      Address to send tokens to.
    /// @param _amount  Amount of tokens to send.
    /// @param _chainId Chain ID of the destination chain.
    function sendERC20(address _to, uint256 _amount, uint256 _chainId) external virtual {
        if (_to == address(0)) revert ZeroAddress();

        _burn(msg.sender, _amount);

        bytes memory _message = abi.encodeCall(this.relayERC20, (msg.sender, _to, _amount));
        IL2ToL2CrossDomainMessenger(MESSENGER).sendMessage(_chainId, address(this), _message);

        emit SendERC20(msg.sender, _to, _amount, _chainId);
    }

    /// @notice Relays tokens received from another chain.
    /// @param _from   Address of the msg.sender of sendERC20 on the source chain.
    /// @param _to     Address to relay tokens to.
    /// @param _amount Amount of tokens to relay.
    function relayERC20(address _from, address _to, uint256 _amount) external virtual {
        if (_to == address(0)) revert ZeroAddress();

        if (msg.sender != MESSENGER) revert CallerNotL2ToL2CrossDomainMessenger();

        if (IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageSender() != address(this)) {
            revert InvalidCrossDomainSender();
        }

        uint256 source = IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageSource();

        _mint(_to, _amount);

        emit RelayERC20(_from, _to, _amount, source);
    }

    /// @notice Returns the name of the token.
    function name() public view virtual override returns (string memory) {
        return _getMetadataStorage().name;
    }

    /// @notice Returns the symbol of the token.
    function symbol() public view virtual override returns (string memory) {
        return _getMetadataStorage().symbol;
    }

    /// @notice Returns the number of decimals used to get its user representation.
    /// For example, if `decimals` equals `2`, a balance of `505` tokens should
    /// be displayed to a user as `5.05` (`505 / 10 ** 2`).
    /// NOTE: This information is only used for _display_ purposes: it in
    /// no way affects any of the arithmetic of the contract, including
    /// {IERC20-balanceOf} and {IERC20-transfer}.
    function decimals() public view virtual override returns (uint8) {
        return _getMetadataStorage().decimals;
    }
}

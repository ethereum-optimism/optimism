// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { ERC20 } from "@solady/tokens/ERC20.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/IL2ToL2CrossDomainMessenger.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @notice Thrown when attempting to relay a message and the function caller (msg.sender) is not
/// L2ToL2CrossDomainMessenger.
error RelayMessageCallerNotL2ToL2CrossDomainMessenger();

/// @notice Thrown when attempting to relay a message and the cross domain message sender is not this
/// OptimismSuperchainERC20.
error MessageSenderNotThisSuperchainERC20();

/// @notice Thrown when attempting to mint or burn tokens and the function caller is not the StandardBridge.
error CallerNotBridge();

/// @notice Thrown when attempting to mint or burn tokens and the account is the zero address.
error ZeroAddress();

/// @custom:proxied
/// @title OptimismSuperchainERC20
/// @notice OptimismSuperchainERC20 is a standard extension of the base ERC20 token contract that unifies ERC20 token
///         bridging to make it fungible across the Superchain. This construction builds on top of the
///         L2ToL2CrossDomainMessenger for both replay protection and domain binding.
contract OptimismSuperchainERC20 is ERC20, ISemver {
    /// @notice Address of the corresponding version of this token on the remote chain.
    address public immutable REMOTE_TOKEN;

    /// @notice Decimals of the token
    uint8 private immutable DECIMALS;

    /// @notice Name of the token
    string private _name;

    /// @notice Symbol of the token
    string private _symbol;

    /// @notice Emitted whenever tokens are minted for an account.
    /// @param account Address of the account tokens are being minted for.
    /// @param amount  Amount of tokens minted.
    event Mint(address indexed account, uint256 amount);

    /// @notice Emitted whenever tokens are burned from an account.
    /// @param account Address of the account tokens are being burned from.
    /// @param amount  Amount of tokens burned.
    event Burn(address indexed account, uint256 amount);

    /// @notice Emitted whenever tokens are sent to another chain.
    /// @param from    Address of the sender.
    /// @param to      Address of the recipient.
    /// @param amount  Amount of tokens sent.
    /// @param chainId Chain ID of the destination chain.
    event SentERC20(address indexed from, address indexed to, uint256 amount, uint256 chainId);

    /// @notice Emitted whenever tokens are successfully relayed on this chain.
    /// @param to     Address of the recipient.
    /// @param amount Amount of tokens relayed.
    event RelayedERC20(address indexed to, uint256 amount);

    /// @notice A modifier that only allows the bridge to call
    modifier onlyBridge() {
        if (msg.sender != Predeploys.L2_STANDARD_BRIDGE) revert CallerNotBridge();
        _;
    }

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.1
    string public constant version = "1.0.0-beta.1";

    /// @param _remoteToken     Address of the corresponding remote token.
    /// @param _tokenName       ERC20 name.
    /// @param _tokenSymbol     ERC20 symbol.
    /// @param _decimals        ERC20 decimals.
    constructor(address _remoteToken, string memory _tokenName, string memory _tokenSymbol, uint8 _decimals) {
        REMOTE_TOKEN = _remoteToken;
        DECIMALS = _decimals;
        _name = _tokenName;
        _symbol = _tokenSymbol;
    }

    /// @notice Allows the StandardBridge to mint tokens.
    /// @param _to     Address to mint tokens to.
    /// @param _amount Amount of tokens to mint.
    function mint(address _to, uint256 _amount) external virtual onlyBridge {
        if (_to == address(0)) revert ZeroAddress();

        _mint(_to, _amount);

        emit Mint(_to, _amount);
    }

    /// @notice Allows the StandardBridge to burn tokens.
    /// @param _from   Address to burn tokens from.
    /// @param _amount Amount of tokens to burn.
    function burn(address _from, uint256 _amount) external virtual onlyBridge {
        if (_from == address(0)) revert ZeroAddress();

        _burn(_from, _amount);

        emit Burn(_from, _amount);
    }

    /// @notice Sends tokens to some target address on another chain.
    /// @param _to      Address to send tokens to.
    /// @param _amount  Amount of tokens to send.
    /// @param _chainId Chain ID of the destination chain.
    function sendERC20(address _to, uint256 _amount, uint256 _chainId) external {
        if (_to == address(0)) revert ZeroAddress();

        _burn(msg.sender, _amount);

        bytes memory _message = abi.encodeCall(this.relayERC20, (_to, _amount));
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).sendMessage(
            _chainId, address(this), _message
        );

        emit SentERC20(msg.sender, _to, _amount, _chainId);
    }

    /// @notice Relays tokens received from another chain.
    /// @param _to     Address to relay tokens to.
    /// @param _amount Amount of tokens to relay.
    function relayERC20(address _to, uint256 _amount) external {
        if (_to == address(0)) revert ZeroAddress();

        if (msg.sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) {
            revert RelayMessageCallerNotL2ToL2CrossDomainMessenger();
        }

        if (
            IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).crossDomainMessageSender()
                != address(this)
        ) {
            revert MessageSenderNotThisSuperchainERC20();
        }

        _mint(_to, _amount);

        emit RelayedERC20(_to, _amount);
    }

    /// @notice Returns the number of decimals used to get its user representation.
    /// For example, if `decimals` equals `2`, a balance of `505` tokens should
    /// be displayed to a user as `5.05` (`505 / 10 ** 2`).
    /// NOTE: This information is only used for _display_ purposes: it in
    /// no way affects any of the arithmetic of the contract, including
    /// {IERC20-balanceOf} and {IERC20-transfer}.
    function decimals() public view override returns (uint8) {
        return DECIMALS;
    }

    /// @notice Returns the name of the token.
    function name() public view virtual override returns (string memory) {
        return _name;
    }

    /// @notice Returns the symbol of the token.
    function symbol() public view virtual override returns (string memory) {
        return _symbol;
    }
}

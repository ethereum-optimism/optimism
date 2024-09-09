// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { IOptimismSuperchainERC20Extension } from "src/L2/interfaces/IOptimismSuperchainERC20.sol";
import { ERC20 } from "@solady/tokens/ERC20.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/interfaces/IL2ToL2CrossDomainMessenger.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Initializable } from "@openzeppelin/contracts-v5/proxy/utils/Initializable.sol";
import { ERC165 } from "@openzeppelin/contracts-v5/utils/introspection/ERC165.sol";

/// @notice Thrown when attempting to relay a message and the function caller (msg.sender) is not
/// L2ToL2CrossDomainMessenger.
error CallerNotL2ToL2CrossDomainMessenger();

/// @notice Thrown when attempting to relay a message and the cross domain message sender is not this
/// OptimismSuperchainERC20.
error InvalidCrossDomainSender();

/// @notice Thrown when attempting to mint or burn tokens and the function caller is not the StandardBridge.
error OnlyBridge();

/// @notice Thrown when attempting to mint or burn tokens and the account is the zero address.
error ZeroAddress();

/// @custom:proxied true
/// @title OptimismSuperchainERC20
/// @notice OptimismSuperchainERC20 is a standard extension of the base ERC20 token contract that unifies ERC20 token
///         bridging to make it fungible across the Superchain. This construction allows the L2StandardBridge to burn
///         and mint tokens. This makes it possible to convert a valid OptimismMintableERC20 token to a SuperchainERC20
///         token, turning it fungible and interoperable across the superchain. Likewise, it also enables the inverse
///         conversion path.
///         Moreover, it builds on top of the L2ToL2CrossDomainMessenger for both replay protection and domain binding.
contract OptimismSuperchainERC20 is IOptimismSuperchainERC20Extension, ERC20, ISemver, Initializable, ERC165 {
    /// @notice Address of the L2ToL2CrossDomainMessenger Predeploy.
    address internal constant MESSENGER = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;

    /// @notice Address of the StandardBridge Predeploy.
    address internal constant BRIDGE = Predeploys.L2_STANDARD_BRIDGE;

    /// @notice Storage slot that the OptimismSuperchainERC20Metadata struct is stored at.
    /// keccak256(abi.encode(uint256(keccak256("optimismSuperchainERC20.metadata")) - 1)) & ~bytes32(uint256(0xff));
    bytes32 internal constant OPTIMISM_SUPERCHAIN_ERC20_METADATA_SLOT =
        0x07f04e84143df95a6373fcf376312ae41da81a193a3089073a54f47a74d8fb00;

    /// @notice Storage struct for the OptimismSuperchainERC20 metadata.
    /// @custom:storage-location erc7201:optimismSuperchainERC20.metadata
    struct OptimismSuperchainERC20Metadata {
        /// @notice Address of the corresponding version of this token on the remote chain.
        address remoteToken;
        /// @notice Name of the token
        string name;
        /// @notice Symbol of the token
        string symbol;
        /// @notice Decimals of the token
        uint8 decimals;
    }

    /// @notice Returns the storage for the OptimismSuperchainERC20Metadata.
    function _getMetadataStorage() private pure returns (OptimismSuperchainERC20Metadata storage _storage) {
        assembly {
            _storage.slot := OPTIMISM_SUPERCHAIN_ERC20_METADATA_SLOT
        }
    }

    /// @notice A modifier that only allows the bridge to call
    modifier onlyBridge() {
        if (msg.sender != BRIDGE) revert OnlyBridge();
        _;
    }

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.2
    string public constant version = "1.0.0-beta.2";

    /// @notice Constructs the OptimismSuperchainERC20 contract.
    constructor() {
        _disableInitializers();
    }

    /// @notice Initializes the contract.
    /// @param _remoteToken    Address of the corresponding remote token.
    /// @param _name           ERC20 name.
    /// @param _symbol         ERC20 symbol.
    /// @param _decimals       ERC20 decimals.
    function initialize(
        address _remoteToken,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        external
        initializer
    {
        OptimismSuperchainERC20Metadata storage _storage = _getMetadataStorage();
        _storage.remoteToken = _remoteToken;
        _storage.name = _name;
        _storage.symbol = _symbol;
        _storage.decimals = _decimals;
    }

    /// @notice Allows the L2StandardBridge to mint tokens.
    /// @param _to     Address to mint tokens to.
    /// @param _amount Amount of tokens to mint.
    function mint(address _to, uint256 _amount) external virtual onlyBridge {
        if (_to == address(0)) revert ZeroAddress();

        _mint(_to, _amount);

        emit Mint(_to, _amount);
    }

    /// @notice Allows the L2StandardBridge to burn tokens.
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

        bytes memory _message = abi.encodeCall(this.relayERC20, (msg.sender, _to, _amount));
        IL2ToL2CrossDomainMessenger(MESSENGER).sendMessage(_chainId, address(this), _message);

        emit SendERC20(msg.sender, _to, _amount, _chainId);
    }

    /// @notice Relays tokens received from another chain.
    /// @param _from   Address of the msg.sender of sendERC20 on the source chain.
    /// @param _to     Address to relay tokens to.
    /// @param _amount Amount of tokens to relay.
    function relayERC20(address _from, address _to, uint256 _amount) external {
        if (_to == address(0)) revert ZeroAddress();

        if (msg.sender != MESSENGER) revert CallerNotL2ToL2CrossDomainMessenger();

        if (IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageSender() != address(this)) {
            revert InvalidCrossDomainSender();
        }

        uint256 source = IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageSource();

        _mint(_to, _amount);

        emit RelayERC20(_from, _to, _amount, source);
    }

    /// @notice Returns the address of the corresponding version of this token on the remote chain.
    function remoteToken() public view override returns (address) {
        return _getMetadataStorage().remoteToken;
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
    function decimals() public view override returns (uint8) {
        return _getMetadataStorage().decimals;
    }

    /// @notice ERC165 interface check function.
    /// @param _interfaceId Interface ID to check.
    /// @return Whether or not the interface is supported by this contract.
    function supportsInterface(bytes4 _interfaceId) public view virtual override returns (bool) {
        return
            _interfaceId == type(IOptimismSuperchainERC20Extension).interfaceId || super.supportsInterface(_interfaceId);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Contracts
import { SuperchainERC20 } from "src/L2/SuperchainERC20.sol";

// Libraries
import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";

// Interfaces
import { ISuperchainERC20Factory } from "interfaces/L2/ISuperchainERC20Factory.sol";

/// @title SuperchainWrappedERC20
/// @notice A SuperchainERC20 representation of an existing ERC20 token, deployed via CREATE2 by the
///         SuperchainERC20Factory. The wrapped token's address is derived solely from the chain ID
///         and address of the original token, so it is identical on every chain in the Superchain.
///         On the original token's chain, the SuperchainERC20Factory mints and burns the wrapped
///         token against escrowed original tokens (wrap/unwrap). On every chain, the
///         SuperchainTokenBridge mints and burns it to move balances across the Superchain.
/// @dev    Deployment parameters are read back from the factory during construction so that the
///         creation code is constant and the CREATE2 address commits only to (chainId, token).
contract SuperchainWrappedERC20 is SuperchainERC20 {
    /// @notice Address of the SuperchainERC20Factory that deployed this token. Only the factory
    ///         can mint and burn through `mint` and `unwrap` flows.
    address public immutable FACTORY;

    /// @notice Address of the original token that this contract wraps.
    address public immutable ORIGINAL_TOKEN;

    /// @notice Chain ID of the chain where the original token lives. Wrapping and unwrapping are
    ///         only possible on that chain; on all other chains supply arrives exclusively through
    ///         the SuperchainTokenBridge.
    uint256 public immutable ORIGINAL_CHAIN_ID;

    /// @notice Decimals of the wrapped token, mirroring the original token.
    uint8 internal immutable WRAPPED_DECIMALS;

    /// @notice Name of the wrapped token.
    string internal wrappedName;

    /// @notice Symbol of the wrapped token.
    string internal wrappedSymbol;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    function version() external pure override returns (string memory) {
        return "1.0.0";
    }

    /// @notice Reads the deployment parameters back from the deploying factory. Constructor takes
    ///         no arguments so that the creation code (and therefore the CREATE2 address) does not
    ///         depend on the token metadata.
    constructor() {
        FACTORY = msg.sender;
        (uint256 chainId, address token, string memory name_, string memory symbol_, uint8 decimals_) =
            ISuperchainERC20Factory(msg.sender).deployConfig();
        ORIGINAL_CHAIN_ID = chainId;
        ORIGINAL_TOKEN = token;
        wrappedName = name_;
        wrappedSymbol = symbol_;
        WRAPPED_DECIMALS = decimals_;
    }

    /// @notice Returns the name of the wrapped token.
    function name() public view override returns (string memory) {
        return wrappedName;
    }

    /// @notice Returns the symbol of the wrapped token.
    function symbol() public view override returns (string memory) {
        return wrappedSymbol;
    }

    /// @notice Returns the decimals of the wrapped token.
    function decimals() public view override returns (uint8) {
        return WRAPPED_DECIMALS;
    }

    /// @notice Allows the SuperchainERC20Factory to mint tokens when original tokens are wrapped.
    /// @param _to     Address to mint tokens to.
    /// @param _amount Amount of tokens to mint.
    function mint(address _to, uint256 _amount) external {
        if (msg.sender != FACTORY) revert Unauthorized();
        _mint(_to, _amount);
    }

    /// @notice Allows the SuperchainERC20Factory to burn tokens when wrapped tokens are unwrapped.
    /// @param _from   Address to burn tokens from.
    /// @param _amount Amount of tokens to burn.
    function burn(address _from, uint256 _amount) external {
        if (msg.sender != FACTORY) revert Unauthorized();
        _burn(_from, _amount);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismMintableERC20 } from "src/universal/OptimismMintableERC20.sol";
import { ISemver } from "src/universal/ISemver.sol";

/// @custom:proxied
/// @custom:predeployed 0x4200000000000000000000000000000000000012
/// @title OptimismMintableERC20Factory
/// @notice OptimismMintableERC20Factory is a factory contract that generates OptimismMintableERC20
///         contracts on the network it's deployed to. Simplifies the deployment process for users
///         who may be less familiar with deploying smart contracts. Designed to be backwards
///         compatible with the older StandardL2ERC20Factory contract.
contract OptimismMintableERC20Factory is ISemver {
    /// @notice Address of the StandardBridge on this chain.
    address public immutable BRIDGE;

    /// @custom:legacy
    /// @notice Emitted whenever a new OptimismMintableERC20 is created. Legacy version of the newer
    ///         OptimismMintableERC20Created event. We recommend relying on that event instead.
    /// @param remoteToken Address of the token on the remote chain.
    /// @param localToken  Address of the created token on the local chain.
    event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken);

    /// @notice Emitted whenever a new OptimismMintableERC20 is created.
    /// @param localToken  Address of the created token on the local chain.
    /// @param remoteToken Address of the corresponding token on the remote chain.
    /// @param deployer    Address of the account that deployed the token.
    event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer);

    /// @notice The semver MUST be bumped any time that there is a change in
    ///         the OptimismMintableERC20 token contract since this contract
    ///         is responsible for deploying OptimismMintableERC20 contracts.
    /// @notice Semantic version.
    /// @custom:semver 1.8.0
    string public constant version = "1.8.0";

    /// @param _bridge Address of the StandardBridge on this chain.
    constructor(address _bridge) {
        BRIDGE = _bridge;
    }

    /// @notice Getter function for the address of the StandardBridge on this chain.
    /// @notice Address of the StandardBridge on this chain.
    function bridge() public view returns (address) {
        return BRIDGE;
    }

    /// @custom:legacy
    /// @notice Creates an instance of the OptimismMintableERC20 contract. Legacy version of the
    ///         newer createOptimismMintableERC20 function, which has a more intuitive name.
    /// @param _remoteToken Address of the token on the remote chain.
    /// @param _name        ERC20 name.
    /// @param _symbol      ERC20 symbol.
    /// @return Address of the newly created token.
    function createStandardL2Token(
        address _remoteToken,
        string memory _name,
        string memory _symbol
    )
        external
        returns (address)
    {
        return createOptimismMintableERC20(_remoteToken, _name, _symbol);
    }

    /// @notice Creates an instance of the OptimismMintableERC20 contract.
    /// @param _remoteToken Address of the token on the remote chain.
    /// @param _name        ERC20 name.
    /// @param _symbol      ERC20 symbol.
    /// @return Address of the newly created token.
    function createOptimismMintableERC20(
        address _remoteToken,
        string memory _name,
        string memory _symbol
    )
        public
        returns (address)
    {
        return createOptimismMintableERC20WithDecimals(_remoteToken, _name, _symbol, 18);
    }

    /// @notice Creates an instance of the OptimismMintableERC20 contract, with specified decimals.
    /// @param _remoteToken Address of the token on the remote chain.
    /// @param _name        ERC20 name.
    /// @param _symbol      ERC20 symbol.
    /// @param _decimals    ERC20 decimals
    /// @return Address of the newly created token.
    function createOptimismMintableERC20WithDecimals(
        address _remoteToken,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        public
        returns (address)
    {
        require(_remoteToken != address(0), "OptimismMintableERC20Factory: must provide remote token address");

        bytes32 salt = keccak256(abi.encode(_remoteToken, _name, _symbol, _decimals));
        address localToken =
            address(new OptimismMintableERC20{ salt: salt }(BRIDGE, _remoteToken, _name, _symbol, _decimals));

        // Emit the old event too for legacy support.
        emit StandardL2TokenCreated(_remoteToken, localToken);

        // Emit the updated event. The arguments here differ from the legacy event, but
        // are consistent with the ordering used in StandardBridge events.
        emit OptimismMintableERC20Created(localToken, _remoteToken, msg.sender);

        return localToken;
    }
}

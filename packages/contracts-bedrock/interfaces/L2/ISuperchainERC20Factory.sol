// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title ISuperchainERC20Factory
/// @notice Interface for the SuperchainERC20Factory contract.
interface ISuperchainERC20Factory {
    error ZeroAddress();
    error Unauthorized();
    error SuperchainERC20Factory_TokenNotDeployed();
    error SuperchainERC20Factory_NoActiveDeployment();
    error SuperchainERC20Factory_InvalidCrossDomainSender();
    error SuperchainERC20Factory_InvalidSourceChain();

    event WrappedTokenDeployed(
        uint256 indexed originalChainId,
        address indexed originalToken,
        address indexed wrappedToken,
        string name,
        string symbol,
        uint8 decimals
    );

    event WrappedTokenPropagated(
        address indexed originalToken, address indexed wrappedToken, uint256 indexed toChainId, bytes32 msgHash
    );

    event Wrapped(address indexed originalToken, address indexed wrappedToken, address indexed sender, uint256 amount);

    event Unwrapped(
        address indexed originalToken, address indexed wrappedToken, address indexed sender, uint256 amount
    );

    function version() external view returns (string memory);

    function deployConfig()
        external
        view
        returns (uint256 chainId_, address token_, string memory name_, string memory symbol_, uint8 decimals_);

    function deploy(
        uint256 _chainId,
        address _token,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        external
        returns (address wrappedToken_);

    function propagate(address _token, uint256 _toChainId) external returns (bytes32 msgHash_);

    function relayDeploy(
        uint256 _chainId,
        address _token,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        external
        returns (address wrappedToken_);

    function wrap(address _token, uint256 _amount) external returns (address wrappedToken_);

    function unwrap(address _token, uint256 _amount) external returns (address wrappedToken_);

    function getWrappedToken(uint256 _chainId, address _token) external view returns (address wrappedToken_);

    function __constructor__() external;
}

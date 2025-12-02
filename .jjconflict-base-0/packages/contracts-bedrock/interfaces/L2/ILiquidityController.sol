// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

interface ILiquidityController is ISemver {
    error LiquidityController_Unauthorized();

    event Initialized(uint8 version);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    event MinterAuthorized(address indexed minter);
    event MinterDeauthorized(address indexed minter);
    event LiquidityMinted(address indexed minter, address indexed to, uint256 amount);
    event LiquidityBurned(address indexed minter, uint256 amount);

    function authorizeMinter(address _minter) external;
    function deauthorizeMinter(address _minter) external;
    function mint(address _to, uint256 _amount) external;
    function burn() external payable;
    function owner() external view returns (address);
    function transferOwnership(address newOwner) external;
    function renounceOwnership() external;
    function minters(address) external view returns (bool);
    function gasPayingTokenName() external view returns (string memory);
    function gasPayingTokenSymbol() external view returns (string memory);
    function initialize(
        address _owner,
        string memory _gasPayingTokenName,
        string memory _gasPayingTokenSymbol
    )
        external;

    function __constructor__() external;
}

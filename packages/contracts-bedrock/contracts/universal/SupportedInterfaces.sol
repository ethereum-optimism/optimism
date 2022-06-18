// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

// Import this here to make it available just by importing this file
import { IERC165 } from "@openzeppelin/contracts/utils/introspection/IERC165.sol";

interface IRemoteToken {
    function mint(address _to, uint256 _amount) external virtual;

    function burn(address _from, uint256 _amount) external virtual;

    function remoteToken() external virtual;
}

interface IL1Token {
    function mint(address _to, uint256 _amount) external virtual;

    function burn(address _from, uint256 _amount) external virtual;

    function l1Token() external virtual;
}

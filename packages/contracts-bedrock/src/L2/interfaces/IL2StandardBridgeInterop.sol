// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { IL2StandardBridge } from "src/L2/interfaces/IL2StandardBridge.sol";

interface IMintableAndBurnable is IERC20 {
    function mint(address, uint256) external;
    function burn(address, uint256) external;
}

interface IL2StandardBridgeInterop is IL2StandardBridge {
    error InvalidDecimals();
    error InvalidLegacyERC20Address();
    error InvalidSuperchainERC20Address();
    error InvalidTokenPair();

    event Converted(address indexed from, address indexed to, address indexed caller, uint256 amount);

    receive() external payable;

    function convert(address _from, address _to, uint256 _amount) external;
}

// SPDX-License-Identifier: MIT
pragma solidity >=0.5.16 <0.8.0;

import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

interface IL2StandardERC20 is IERC20 {
    function l1Token() external returns (address);

    function mint(address _to, uint256 _value) external;

    function burn(address _from, uint256 _value) external;
}

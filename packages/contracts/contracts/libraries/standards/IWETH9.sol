// SPDX-License-Identifier: GPL-2.0-or-later
pragma solidity =0.7.6;
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/// @title Interface for WETH9. Also contains the non-ERC20 events
/// normally present in the WETH9 implementation.
interface IWETH9 is IERC20 {
    event Deposit(address indexed dst, uint256 wad);
    event Withdrawal(address indexed src, uint256 wad);

    /// @notice Deposit ether to get wrapped ether
    function deposit() external payable;

    /// @notice Withdraw wrapped ether to get ether
    function withdraw(uint256) external;
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

import { console } from "forge-std/console.sol";

interface IAllowanceTransfer {
    /**
     * @notice Transfer approved tokens from one address to another
     * @param from   The address to transfer from
     * @param to     The address of the recipient
     * @param amount The amount of the token to transfer
     * @param token  The token address to transfer
     * @notice Requires the from address to have approved at least the desired amount
     *         of tokens to msg.sender.
     */
    function transferFrom(address from, address to, uint160 amount, address token) external;

    /**
     * @notice Approves the spender to use up to amount of the specified token up until the expiration
     * @param token The token to approve
     * @param spender The spender address to approve
     * @param amount The approved amount of the token
     * @param expiration The timestamp at which the approval is no longer valid
     * @notice The packed allowance also holds a nonce, which will stay unchanged in approve
     * @notice Setting amount to type(uint160).max sets an unlimited approval
     */
    function approve(address token, address spender, uint160 amount, uint48 expiration) external;

    /**
     * @notice Retrieves the allowances. Returns the allowed amount, expiration at which the
     *         allowed amount is no longer valid, and current nonce thats updated on any
     *         signature based approvals.
     */
    function allowance(address owner, address token, address spender) external view returns (uint160, uint48, uint48);
}

/**
 * @title Permit2Lib
 * @notice Enables efficient transfers for any token by falling back to Permit2.
 */
library Permit2Lib {
    /**
     * @notice The address of the Permit2 contract the library will use.
     */
    IAllowanceTransfer internal constant PERMIT2 =
        IAllowanceTransfer(address(0x000000000022D473030F116dDEE9F6B43aC78BA3));

    /**
     * @notice Transfer a given amount of tokens from one user to another.
     * @param _token The token to transfer.
     * @param _from The user to transfer from.
     * @param _to The user to transfer to.
     * @param _amount The amount to transfer.
     */
    function transferFrom2(IERC20 _token, address _from, address _to, uint256 _amount) internal {
        // Generate calldata for a standard transferFrom call.
        bytes memory inputData = abi.encodeCall(IERC20.transferFrom, (_from, _to, _amount));

        bool success; // Call the token contract as normal, capturing whether it succeeded.
        assembly {
            success :=
                and(
                    // Set success to whether the call reverted, if not we check it either
                    // returned exactly 1 (can't just be non-zero data), or had no return data.
                    or(eq(mload(0), 1), iszero(returndatasize())),
                    // Counterintuitively, this call() must be positioned after the or() in the
                    // surrounding and() because and() evaluates its arguments from right to left.
                    // We use 0 and 32 to copy up to 32 bytes of return data into the first slot of scratch space.
                    call(gas(), _token, 0, add(inputData, 32), mload(inputData), 0, 32)
                )
        }

        require(_amount <= type(uint160).max, "Permit2Lib: value larger than uint160");

        // We'll fall back to using Permit2 if calling transferFrom on the token directly reverted.
        if (!success) PERMIT2.transferFrom(_from, _to, uint160(_amount), address(_token));
    }
}

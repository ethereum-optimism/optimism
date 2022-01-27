// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title IL1NativeERC20Bridge
 */
interface IL1NativeERC20Bridge {
    /**********
     * Events *
     **********/

    event NativeERC20WithdrawalInitiated(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    event NativeERC20DepositFinalized(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    event NativeERC20DepositFailed(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    /********************
     * Public Functions *
     ********************/

    /**
     * @dev get the address of the corresponding L1 bridge contract.
     * @return Address of the corresponding L1 bridge contract.
     */
    function l2TokenBridge() external returns (address);

    /**
     * @dev initiate a withdraw of some tokens to the caller's account on L2
     * @param _l1Token Address of L1 token where withdrawal was initiated.
     * @param _amount Amount of the token to withdraw.
     * param _l2Gas Unused, but included for potential forward compatibility considerations.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function withdraw(
        address _l1Token,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    ) external;

    /**
     * @dev Initiate a withdraw of some token to a recipient's account on L2.
     * @param _l1Token Address of L1 token where withdrawal is initiated.
     * @param _to L2 address to credit the withdrawal to.
     * @param _amount Amount of the token to withdraw.
     * param _l2Gas Unused, but included for potential forward compatibility considerations.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function withdrawTo(
        address _l1Token,
        address _to,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    ) external;

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * @dev Complete a deposit from L2 to L1, and credits funds to the recipient's balance of this
     * L1 token. This call will fail if it did not originate from a corresponding deposit in
     * L2NativeERC20Bridge.
     * @param _l2Token Address for the l2 token this is called with
     * @param _l1Token Address for the l1 token this is called with
     * @param _from Account to pull the deposit from on L2.
     * @param _to Address to receive the withdrawal at on L1
     * @param _amount Amount of the token to withdraw
     * @param _data Data provider by the sender on L1. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function finalizeDeposit(
        address _l2Token,
        address _l1Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) external;
}

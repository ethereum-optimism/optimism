// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.9.0;

/**
 * @title IL2NativeERC20Bridge
 */
interface IL2NativeERC20Bridge {
    /**********
     * Events *
     **********/

    event NativeERC20DepositInitiated(
        address indexed _l2Token,
        address indexed _l1Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    event NativeERC20WithdrawalFinalized(
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
     * @dev get the address of the corresponding L1 native bridge contract.
     * @return Address of the corresponding L1 native bridge contract.
     */
    function l1TokenBridge() external returns (address);

    /**
     * @dev deposit an amount of the ERC20 to the caller's balance on L1.
     * @param _l2Token Address of the L2 ERC20 we are depositing
     * @param _l1Token Address of the L2 respective L1 ERC20
     * @param _amount Amount of the ERC20 to deposit
     * @param _l1Gas Gas limit required to complete the deposit on L1.
     * @param _data Optional data to forward to L1. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function depositERC20(
        address _l2Token,
        address _l1Token,
        uint256 _amount,
        uint32 _l1Gas,
        bytes calldata _data
    ) external;

    /**
     * @dev deposit an amount of ERC20 to a recipient's balance on L2.
     * @param _l2Token Address of the L2 ERC20 we are depositing
     * @param _l1Token Address of the L1 respective L2 ERC20
     * @param _to L2 address to credit the withdrawal to.
     * @param _amount Amount of the ERC20 to deposit.
     * @param _l1Gas Gas limit required to complete the deposit on L1.
     * @param _data Optional data to forward to L1. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function depositERC20To(
        address _l2Token,
        address _l1Token,
        address _to,
        uint256 _amount,
        uint32 _l1Gas,
        bytes calldata _data
    ) external;

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * @dev Complete a withdrawal from L1 to L2, and credit funds to the recipient's balance of the
     * L2 ERC20 token.
     * This call will fail if the initialized withdrawal from L1 has not been finalized.
     *
     * @param _l2Token Address of L2 token to finalizeWithdrawal for.
     * @param _l1Token Address of L1 token where withdrawal was initiated.
     * @param _from L1 address initiating the transfer.
     * @param _to L2 address to credit the withdrawal to.
     * @param _amount Amount of the ERC20 to deposit.
     * @param _data Data provided by the sender on L1. This data is provided
     *   solely as a convenience for external contracts. Aside from enforcing a maximum
     *   length, these contracts provide no guarantees about its content.
     */
    function finalizeERC20Withdrawal(
        address _l2Token,
        address _l1Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) external;
}

//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/**
 * @title Withdrawor
 */
contract Withdrawor {
    uint256 public nonce;
    mapping(bytes32 => bool) public withdrawals;

    event WithdrawalInitiated(
        uint256 indexed messageNonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes message
    );

    /**
     * Passes a message to L1.
     * @param _message Message to pass to L1.
     */
    function initiateWithdrawal(
        address _target,
        uint256 _gasLimit,
        bytes calldata _message
    ) external payable {
        bytes32 messageHash = keccak256(
            abi.encode(nonce, msg.sender, _target, msg.value, _message)
        );
        withdrawals[messageHash] = true;
        nonce++;

        emit WithdrawalInitiated(nonce, msg.sender, _target, msg.value, _gasLimit, _message);
    }
}

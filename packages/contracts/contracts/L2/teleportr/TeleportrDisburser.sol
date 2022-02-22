// SPDX-License-Identifier: MIT
pragma solidity >=0.8.9;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title TeleportrDisburser
 */
contract TeleportrDisburser is Ownable {
    /**
     * @notice A struct holding the address and amount to disbursement.
     */
    struct Disbursement {
        uint256 amount;
        address addr;
    }

    /// The total number of disbursements processed.
    uint256 public totalDisbursements;

    /**
     * @notice Emitted any time the balance is withdrawn by the owner.
     * @param owner The current owner and recipient of the funds.
     * @param balance The current contract balance paid to the owner.
     */
    event BalanceWithdrawn(address indexed owner, uint256 balance);

    /**
     * @notice Emitted any time a disbursement is successfuly sent.
     * @param depositId The unique sequence number identifying the deposit.
     * @param to The recipient of the disbursement.
     * @param amount The amount sent to the recipient.
     */
    event DisbursementSuccess(uint256 indexed depositId, address indexed to, uint256 amount);

    /**
     * @notice Emitted any time a disbursement fails to send.
     * @param depositId The unique sequence number identifying the deposit.
     * @param to The intended recipient of the disbursement.
     * @param amount The amount intended to be sent to the recipient.
     */
    event DisbursementFailed(uint256 indexed depositId, address indexed to, uint256 amount);

    /**
     * @notice Initializes a new TeleportrDisburser contract.
     */
    constructor() {
        totalDisbursements = 0;
    }

    /**
     * @notice Accepts a list of Disbursements and forwards the amount paid to
     * the contract to each recipient. The method reverts if there are zero
     * disbursements, the total amount to forward differs from the amount sent
     * in the transaction, or the _nextDepositId is unexpected. Failed
     * disbursements will not cause the method to revert, but will instead be
     * held by the contract and availabe for the owner to withdraw.
     * @param _nextDepositId The depositId of the first Dispursement.
     * @param _disbursements A list of Disbursements to process.
     */
    function disburse(uint256 _nextDepositId, Disbursement[] calldata _disbursements)
        external
        payable
        onlyOwner
    {
        // Ensure there are disbursements to process.
        uint256 _numDisbursements = _disbursements.length;
        require(_numDisbursements > 0, "No disbursements");

        // Ensure the _nextDepositId matches our expected value.
        uint256 _depositId = totalDisbursements;
        require(_depositId == _nextDepositId, "Unexpected next deposit id");
        unchecked {
            totalDisbursements += _numDisbursements;
        }

        // Ensure the amount sent in the transaction is equal to the sum of the
        // disbursements.
        uint256 _totalDisbursed = 0;
        for (uint256 i = 0; i < _numDisbursements; i++) {
            _totalDisbursed += _disbursements[i].amount;
        }
        require(_totalDisbursed == msg.value, "Disbursement total != amount sent");

        // Process disbursements.
        for (uint256 i = 0; i < _numDisbursements; i++) {
            uint256 _amount = _disbursements[i].amount;
            address _addr = _disbursements[i].addr;

            // Deliver the dispursement amount to the receiver. If the
            // disbursement fails, the amount will be kept by the contract
            // rather than reverting to prevent blocking progress on other
            // disbursements.

            // slither-disable-next-line calls-loop,reentrancy-events
            (bool success, ) = _addr.call{ value: _amount, gas: 2300 }("");
            if (success) emit DisbursementSuccess(_depositId, _addr, _amount);
            else emit DisbursementFailed(_depositId, _addr, _amount);

            unchecked {
                _depositId += 1;
            }
        }
    }

    /**
     * @notice Sends the contract's current balance to the owner.
     */
    function withdrawBalance() external onlyOwner {
        address _owner = owner();
        uint256 balance = address(this).balance;
        emit BalanceWithdrawn(_owner, balance);
        payable(_owner).transfer(balance);
    }
}

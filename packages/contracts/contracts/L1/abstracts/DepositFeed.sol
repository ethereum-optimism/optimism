//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/* Library Imports */
import {
    AddressAliasHelper
} from "../../../lib/optimism/packages/contracts/contracts/standards/AddressAliasHelper.sol";

/**
 * @title DepositFeed
 * @notice Implements the logic for depositing from L1 to L2.
 */
abstract contract DepositFeed {
    /**********
     * Errors *
     **********/

    /**
     * @notice Error emitted on deposits which create a new contract with a non-zero target.
     */
    error NonZeroCreationTarget();

    /**********
     * Events *
     **********/

    /**
     * @notice Emitted when a Transaction is deposited from L1 to L2. The parameters of this
     * event are read by the rollup node and used to derive deposit transactions on L2.
     */
    event TransactionDeposited(
        address indexed from,
        address indexed to,
        uint256 mint,
        uint256 value,
        uint256 gasLimit,
        bool isCreation,
        bytes data
    );

    /**********************
     * External Functions *
     **********************/

    /**
     * @notice Accepts deposits of ETH and data, and emits a TransactionDeposited event for use in
     * deriving deposit transactions.
     * @param _to The L2 destination address.
     * @param _value The ETH value to send in the deposit transaction.
     * @param _gasLimit The L2 gasLimit.
     * @param _isCreation Whether or not the transaction should be contract creation.
     * @param _data The input data.
     */
    function depositTransaction(
        address _to,
        uint256 _value,
        uint256 _gasLimit,
        bool _isCreation,
        bytes memory _data
    ) public payable {
        if (_isCreation && _to != address(0)) {
            revert NonZeroCreationTarget();
        }

        address from = msg.sender;
        // Transform the from-address to its alias if the caller is a contract.
        if (msg.sender != tx.origin) {
            from = AddressAliasHelper.applyL1ToL2Alias(msg.sender);
        }

        emit TransactionDeposited(from, _to, msg.value, _value, _gasLimit, _isCreation, _data);
    }
}

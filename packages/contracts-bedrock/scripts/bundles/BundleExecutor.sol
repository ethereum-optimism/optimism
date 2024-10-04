// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CREATE3 } from "lib/solady/src/utils/CREATE3.sol";

struct BundleTransaction {
    address to;
    uint256 value;
    bytes data;
}

contract BundleExecutor {
    /// @notice Address to send any ETH to after executing the transactions.
    address public immutable beneficiary;

    /// @param _transactions Transactions to execute.
    /// @param _beneficiary Address to send any ETH to after executing the transactions.
    constructor(BundleTransaction[] memory _transactions, address _beneficiary) payable {
        beneficiary = _beneficiary;
        for (uint256 i = 0; i < _transactions.length; i++) {
            BundleTransaction memory transaction = _transactions[i];
            (bool success,) = transaction.to.call{ value: transaction.value }(transaction.data);
            if (!success) {
                revert("BundleExecutor: transaction failed");
            }
        }
    }

    function destroy() public {
        selfdestruct(payable(beneficiary));
    }
}

contract BundleExecutorFactory {
    function execute(string memory _salt, BundleTransaction[] memory _transactions) public payable {
        BundleExecutor executor = BundleExecutor(CREATE3.deploy(
            keccak256(abi.encode(msg.sender, _salt)),
            abi.encodePacked(type(BundleExecutor).creationCode, abi.encode(_transactions, msg.sender)),
            msg.value
        ));
        executor.destroy();
    }

    function predict(string memory _salt) public view returns (address) {
        return CREATE3.getDeployed(keccak256(abi.encode(msg.sender, _salt)), address(this));
    }
}

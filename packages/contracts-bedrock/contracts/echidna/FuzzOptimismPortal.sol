pragma solidity 0.8.15;

import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";

contract EchidnaFuzzOptimismPortal is OptimismPortal {
    uint256 reinitializedCount;
    bool failedDepositCreationNonZeroAddr;

    constructor() OptimismPortal(L2OutputOracle(address(0)), 10) {
        // Note: The base constructor will call initialize() once here.
    }

    /**
     * @notice This method calls upon OptimismPortal.initialize() to ensure
     * no additional initializations are possible.
     */
    function initializeEx() public {
        super.initialize();
        reinitializedCount++;
    }

    /**
     * @notice This method calls upon OptimismPortal.depositTransaction() to ensure
     * a deposit with _isCreation set to true never succeeds with a _to address that
     * has a non-zero value.
     */
    function depositTransactionIsCreation(
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bytes memory _data
    ) public {
        // Deposit with our given fuzz parameters and _isCreation set to true
        depositTransaction(_to, _value, _gasLimit, true, _data);

        // If we did not revert and our _to address is not zero, flag a failure.
        if (_to != address(0x0)) {
            failedDepositCreationNonZeroAddr = true;
        }
    }

    function echidna_never_initialize_twice() public view returns (bool) {
        return reinitializedCount == 0;
    }

    function echidna_never_nonzero_to_creation_deposit() public view returns (bool) {
        return !failedDepositCreationNonZeroAddr;
    }
}

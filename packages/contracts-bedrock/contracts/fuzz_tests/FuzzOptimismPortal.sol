import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";

contract FuzzOptimismPortal is OptimismPortal {
    uint reinitializedCount;
    bool failedDepositCreationNonZeroAddr;
    bool failedAliasingContractFromAddr;
    bool failedNoAliasingFromEOA;
    bool failedMintedLessThanTaken;


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
    function depositTransactionIsCreation(address _to, uint256 _value, uint64 _gasLimit, bytes memory _data) public {
        // Deposit with our given fuzz parameters and _isCreation set to true
        depositTransaction(_to, _value, _gasLimit, true, _data);

        // If we did not revert and our _to address is not zero, flag a failure.
        if (_to != address(0x0))
        {
            failedDepositCreationNonZeroAddr = true;
        }
    }

    /**
     * @notice This method calls upon OptimismPortal.depositTransaction() from a
     * contract address (itself, it performs an external call) to ensure contract
     * aliasing is tested by depositTransactionTestInternal.
     */
    function depositTransactionFromContract(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes memory _data) public payable {
        // We perform an external call to our own address to trigger the conditions for a deposit by a contract address.
        // This is because when we perform an external call, the receiving function will see msg.sender as the caller's address.
        // Because we provide a function to ensure a call from a contract address, we'll be sure the fuzzer tested
        // this case.
        OptimismPortal(payable(this)).depositTransaction{value: msg.value}(_to, _value, _gasLimit, _isCreation, _data);
    }

    /**
     * @notice This override is called at the end of OptimismPortal.depositTransaction()
     * so that we can sanity check all of the input and omitted data.
     */
    function depositTransactionTestInternal(
        address from,
        address to,
        uint256 version,
        uint256 mintValue,
        uint256 sendValue,
        uint64 gasLimit,
        bool isCreation,
        bytes memory data
    ) override internal {
        // Check if the caller is a contract and confirm our address aliasing properties
        if (msg.sender != tx.origin) {
            // If the caller is a contract, we expect the address to be aliased.
            if(AddressAliasHelper.undoL1ToL2Alias(from) != msg.sender) {
                failedAliasingContractFromAddr = true;
            }
        } else {
            // If the caller is an EOA address, we expect the address not to be aliased.
            if (from != msg.sender) {
                failedNoAliasingFromEOA = true;
            }
        }

        // If our mint value exceeds the amount paid, we failed a test.
        if (mintValue > msg.value) {
            failedMintedLessThanTaken = true;
        }
    }

    function echidna_never_initialize_twice() public view returns (bool) {
        return reinitializedCount == 0;
    }

    function echidna_never_nonzero_to_creation_deposit() public view returns (bool) {
        return !failedDepositCreationNonZeroAddr;
    }

    function echidna_alias_from_contract_deposit() public view returns (bool) {
        return !failedAliasingContractFromAddr;
    }

    function echidna_no_alias_from_EOA_deposit() public view returns (bool) {
        return !failedNoAliasingFromEOA;
    }

    function echidna_mint_less_than_taken() public view returns (bool) {
        return !failedMintedLessThanTaken;
    }


}
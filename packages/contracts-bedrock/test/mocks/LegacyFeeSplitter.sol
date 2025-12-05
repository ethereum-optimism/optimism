// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Predeploys } from "src/libraries/Predeploys.sol";

interface ILegacyFeeVault {
    function withdraw() external;
}

/// @title LegacyFeeSplitter
/// @notice A simple contract meant to be used to withdraw fees from FeeVault(s) using the old interface.
/// @dev This contract is not used for production, but rather for testing purposes as it lacks safe guards and error
/// handling.
contract LegacyFeeSplitter {
    receive() external payable { }

    function disburseFees() external {
        _feeVaultWithdrawal(payable(Predeploys.SEQUENCER_FEE_WALLET));
        _feeVaultWithdrawal(payable(Predeploys.BASE_FEE_VAULT));
        _feeVaultWithdrawal(payable(Predeploys.L1_FEE_VAULT));
        _feeVaultWithdrawal(payable(Predeploys.OPERATOR_FEE_VAULT));
    }

    function _feeVaultWithdrawal(address payable _feeVault) internal {
        bytes memory _calldata = abi.encodeCall(ILegacyFeeVault.withdraw, ());
        (bool success,) = _feeVault.call(_calldata);
        require(success, "LegacyFeeSplitter: fee vault withdrawal failed");
    }
}

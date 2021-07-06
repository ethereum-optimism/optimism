// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/**
 * @title Lib_ErrorUtils
 */
library Lib_ErrorUtils {

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Encodes an error string into raw solidity-style revert data.
     * (i.e. ascii bytes, prefixed with bytes4(keccak("Error(string))"))
     * Ref: https://docs.soliditylang.org/en/v0.8.2/control-structures.html?highlight=Error(string)
     * #panic-via-assert-and-error-via-require
     * @param _reason Reason for the reversion.
     * @return Standard solidity revert data for the given reason.
     */
    function encodeRevertString(
        string memory _reason
    )
        internal
        pure
        returns (
            bytes memory
        )
    {
        return abi.encodeWithSignature(
            "Error(string)",
            _reason
        );
    }
}

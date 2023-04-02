// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/**
 * @title SafeCall
 * @notice Perform low level safe calls
 */
library SafeCall {
    /**
     * @notice Perform a low level call without copying any returndata
     *
     * @param _target   Address to call
     * @param _gas      Amount of gas to pass to the call
     * @param _value    Amount of value to pass to the call
     * @param _calldata Calldata to pass to the call
     */
    function call(
        address _target,
        uint256 _gas,
        uint256 _value,
        bytes memory _calldata
    ) internal returns (bool) {
        bool _success;
        assembly {
            _success := call(
                _gas, // gas
                _target, // recipient
                _value, // ether value
                add(_calldata, 32), // inloc
                mload(_calldata), // inlen
                0, // outloc
                0 // outlen
            )
        }
        return _success;
    }

    /**
     * @notice Perform a low level call without copying any returndata. This function
     *         will revert if the call cannot be performed with the specified minimum
     *         gas.
     *
     * @param _target   Address to call
     * @param _minGas   The minimum amount of gas that may be passed to the call
     * @param _value    Amount of value to pass to the call
     * @param _calldata Calldata to pass to the call
     */
    function callWithMinGas(
        address _target,
        uint256 _minGas,
        uint256 _value,
        bytes memory _calldata
    ) internal returns (bool) {
        bool _success;
        assembly {
            // Assertion: gasleft() >= ((_minGas + 200) * 64) / 63
            //
            // Because EIP-150 ensures that, a maximum of 63/64ths of the remaining gas in the call
            // frame may be passed to a subcontext, we need to ensure that the gas will not be
            // truncated to hold this function's invariant: "If a call is performed by
            // `callWithMinGas`, it must receive at least the specified minimum gas limit." In
            // addition, exactly 51 gas is consumed between the below `GAS` opcode and the `CALL`
            // opcode, so it is factored in with some extra room for error.
            if lt(gas(), div(mul(64, add(_minGas, 200)), 63)) {
                // Store the "Error(string)" selector in scratch space.
                mstore(0, 0x08c379a0)
                // Store the pointer to the string length in scratch space.
                mstore(32, 32)
                // Store the string.
                //
                // SAFETY:
                // - We pad the beginning of the string with two zero bytes as well as the
                // length (24) to ensure that we override the free memory pointer at offset
                // 0x40. This is necessary because the free memory pointer is likely to
                // be greater than 1 byte when this function is called, but it is incredibly
                // unlikely that it will be greater than 3 bytes. As for the data within
                // 0x60, it is ensured that it is 0 due to 0x60 being the zero offset.
                // - It's fine to clobber the free memory pointer, we're reverting.
                mstore(88, 0x0000185361666543616c6c3a204e6f7420656e6f75676820676173)

                // Revert with 'Error("SafeCall: Not enough gas")'
                revert(28, 100)
            }

            // The call will be supplied at least (((_minGas + 200) * 64) / 63) - 49 gas due to the
            // above assertion. This ensures that, in all circumstances, the call will
            // receive at least the minimum amount of gas specified.
            // We can prove this property by solving the inequalities:
            // ((((_minGas + 200) * 64) / 63) - 49) >= _minGas
            // ((((_minGas + 200) * 64) / 63) - 51) * (63 / 64) >= _minGas
            // Both inequalities hold true for all possible values of `_minGas`.
            _success := call(
                gas(), // gas
                _target, // recipient
                _value, // ether value
                add(_calldata, 32), // inloc
                mload(_calldata), // inlen
                0x00, // outloc
                0x00 // outlen
            )
        }
        return _success;
    }
}

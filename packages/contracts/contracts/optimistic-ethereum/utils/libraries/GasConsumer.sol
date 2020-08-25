pragma solidity ^0.5.0;

contract GasConsumer {
    // default fallback--consumes all allotted gas
    function() external {
        assembly {
            invalid()
        }
    }

    // Overhead for checking methodId etc in this function before the actual call()
    // This was figured out empirically during testing--adding methods or changing compiler settings will require recalibration.
    uint constant gasOverheadOfEOACall = 947;

    /**
     * Consumes the exact amount of gas specified by the input.
     * Does not include the cost of calling this contract itself, so is best used for testing/entry point calls.
     * @param _amount Amount of gas to consume.
     */
    function consumeGasEOA(uint _amount) external {
        require(_amount > gasOverheadOfEOACall, "Unable to consume an amount of gas this small.");
        uint gasToAlloc = _amount - gasOverheadOfEOACall;
        // call this contract's fallback which consumes all allocated gas
        assembly {
            pop(call(gasToAlloc, address, 0, 0, 0, 0, 0))
        }
    }

    // Overhead for checking methodId, etc. in this function before the actual call()
    // This was figured out empirically during testing--adding methods or changing compiler settings will require recalibration.
    uint constant gasOverheadOfInternalCall = 2514;

    /**
     * Consumes the exact amount of gas specified by the input.
     * Includes the additional cost of CALLing this contract itself, so is best used for cross-contract calls.
     * @param _amount Amount of gas to consume.
     */
    function consumeGasInternalCall(uint _amount) external {
        require(_amount > gasOverheadOfInternalCall, "Unable to consume an amount of gas this small.");
        uint gasToAlloc = _amount - gasOverheadOfInternalCall;
        // call this contract's fallback which consumes all allocated gas
        assembly {
            pop(call(gasToAlloc, address, 0, 0, 0, 0, 0))
        }
    }
}
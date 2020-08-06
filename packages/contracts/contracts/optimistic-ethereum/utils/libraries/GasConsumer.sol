pragma solidity ^0.5.0;

contract GasConsumer {
    // default fallback--consumes all allotted gas
    function() external {
        uint i;
        while (true) {
            i += 420;
        }
    }

    // Overhead for checking methodId etc in this function before the actual call()
    // This was figured out empirically during testing.
    uint constant constantOverheadEOA = 947;

    /**
     * Consumes the exact amount of gas specified by the input.
     * Does not include the cost of calling this contract itself, so is best used for testing/entry point calls.
     * @param _amount Amount of gas to consume.
     */
    function consumeGasEOA(uint _amount) external {
        require(_amount > constantOverheadEOA, "Unable to consume an amount of gas this small.");
        uint gasToAlloc = _amount - constantOverheadEOA;
        // call this contract's fallback which consumes all allocated gas
        assembly {
            pop(call(gasToAlloc, address, 0, 0, 0, 0, 0))
        }
    }

    // Overhead for checking methodId, etc. in this function before the actual call()
    // This was figured out empirically during testing.
    uint constant constantOverheadInternal = 2514;

    /**
     * Consumes the exact amount of gas specified by the input.
     * Includes the additional cost of CALLing this contract itself, so is best used for cross-contract calls.
     * @param _amount Amount of gas to consume.
     */
    function consumeGasInternalCall(uint _amount) external {
        require(_amount > constantOverheadInternal, "Unable to consume an amount of gas this small.");
        uint gasToAlloc = _amount - constantOverheadInternal;
        // call this contract's fallback which consumes all allocated gas
        assembly {
            pop(call(gasToAlloc, address, 0, 0, 0, 0, 0))
        }
    }
}
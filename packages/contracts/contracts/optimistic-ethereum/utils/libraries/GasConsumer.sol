pragma solidity ^0.5.0;

contract GasConsumer {

    // default fallback consumes all allotted gas
    function() external {
        uint i;
        while (true) {
            i += 420;
        }
    }

    // Overhead for checking methodId etc in this function before the actual call()--This was figured out empirically during testing.
    uint constant constantOverhead = 947;
    function consumeGas(uint _amount) external {
        uint gasToAlloc = _amount - constantOverhead;
        // call this contract's fallback which consumes all allocated gas
        assembly {
            pop(call(gasToAlloc, address, 0, 0, 0, 0, 0))
        }
    }
}
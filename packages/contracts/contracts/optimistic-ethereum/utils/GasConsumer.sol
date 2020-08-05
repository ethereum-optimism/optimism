pragma solidity ^0.5.0;

contract GasConsumer {

    // default fallback should consume all allotted gas
    function() external {
        uint i;
        while (true) {
            i++;
        }
    }

    function consumeGasExceeding(uint _amount) public view {
        uint startGas = gasleft();
        while (true) {
            if (startGas - gasleft() > _amount) {
                return;
            }
        }
    }

    // overhead for checking methodId etc in this function before the actual call().
    // This was figured out empirically during testing.
    uint constant constantOverhead = 969;
    function consumeGasExact(uint _amount) external {
        uint gasToAlloc = _amount - constantOverhead;
        // call our fallback which consumes all allocated gas
        assembly {
            pop(call(gasToAlloc, address, 0, 0, 0, 0, 0))
        }
    }
}
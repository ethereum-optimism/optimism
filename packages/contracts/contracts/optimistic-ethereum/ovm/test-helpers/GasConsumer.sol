pragma solidity ^0.5.0;

contract GasConsumer {
    function consumeGasExceeding(uint _amount) public view {
        uint startGas = gasleft();
        while (true) {
            if (startGas - gasleft() > _amount) {
                return;
            }
        }
    }
}
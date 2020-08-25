pragma solidity ^0.5.0;

/* Contract Imports */
import { GasConsumer } from "../optimistic-ethereum/utils/libraries/GasConsumer.sol";


contract GasConsumerCaller {
    function getGasConsumedByGasConsumer(address _consumer, uint _amount) public returns(uint) {
        uint theGas = gasleft();
        GasConsumer(_consumer).consumeGasInternalCall(_amount);
        theGas -= gasleft();
        return theGas;
    }
}
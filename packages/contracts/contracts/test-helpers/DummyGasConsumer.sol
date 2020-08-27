pragma solidity ^0.5.0;

/* Contract Imports */
import { GasConsumer } from "../optimistic-ethereum/utils/libraries/GasConsumer.sol";

/* Testing Imports */
import { console } from "@nomiclabs/buidler/console.sol";

contract DummyGasConsumer {
    GasConsumer private gasConsumer;
    uint private amountOfGasToConsume;
    constructor() public {
        gasConsumer = new GasConsumer();
    }

    function () external {
        gasConsumer.consumeGasInternalCall(amountOfGasToConsume);
    }

    function setAmountGasToConsume(uint _amount) external {
        amountOfGasToConsume = _amount;
    }
}
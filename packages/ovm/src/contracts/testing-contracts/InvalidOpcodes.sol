pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

contract InvalidOpcodes {
    function getCoinbase() public returns (address){
        return block.coinbase;
    }

    function getDifficulty() public returns (uint){
        return block.difficulty;
    }

    function getBlockNumber() public returns (uint) {
        return block.number;
    }
}

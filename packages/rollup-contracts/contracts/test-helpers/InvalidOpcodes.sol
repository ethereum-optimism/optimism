pragma solidity ^0.5.0;

contract InvalidOpcodes {
    function getCoinbase() public view returns (address){
        return block.coinbase;
    }

    function getDifficulty() public view returns (uint){
        return block.difficulty;
    }

    function getBlockNumber() public view returns (uint) {
        return block.number;
    }
}

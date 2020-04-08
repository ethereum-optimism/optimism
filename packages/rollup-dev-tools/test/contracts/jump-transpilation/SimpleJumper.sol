pragma solidity ^0.5.0;

contract SimpleJumper {
    mapping(uint256 => uint256) public times;
    function staticIfTrue() public returns(bool) {
        if (true) {
            times[block.timestamp] = 15;
            return(true);
        } else {
            return(false);
        }
    }

    function staticIfFalseElse() public returns(bool) {
        if (false) {
            return(true);
        } else {
            times[block.timestamp] = 15;
            return(false);
        }
    }

    function doForLoop() public returns(uint256) {
        uint256 val = 29;
        for (uint i=0; i<25; i++) {
            times[block.timestamp] = 15;
            val = val + 7*i;
        }
        return val;
    }

    function doWhileLoop() public returns(uint256) { 
        uint256 val = 29;
        while(val <= 200){
            times[block.timestamp] = 15;
            val = val + 7;
        }
        return val;
    }

    function doLoopingSubcalls() public returns(uint256) {
        uint256 val = 29;
        while(val <= 20000){
            times[block.timestamp] = 15;
            val = val + doForLoop();
        }
        return val;
    }

    function doCrazyCombination(uint256 _input) public returns(uint256) {
        times[block.timestamp] = 15;
        if (_input == 0) {
            times[block.timestamp] = 15;
            return 0;
        } else {
        uint256 val = 29;
        for (uint i=0; i<8; i++) {
            times[block.timestamp] = doLoopingSubcalls();
            val = val + 7*i;
        }
        return val;
        }

    }
}
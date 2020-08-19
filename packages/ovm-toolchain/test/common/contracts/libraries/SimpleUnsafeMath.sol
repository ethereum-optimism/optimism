pragma solidity ^0.5.0;

library SimpleUnsafeMath {
    
    function subUint(uint a, uint b) public  returns(uint){
        
        return a - b;
        
    }
    function addUint(uint a , uint b) public pure returns(uint){        
        uint c = a + b;
        return c;
    }
}
pragma solidity ^0.5.0;

library SimpleSafeMath {
    
    function subUint(uint a, uint b) public  returns(uint){
        
        require(a >= b); // Make sure it doesn't return a negative value.
        return a - b;
        
    }
    function addUint(uint a , uint b) public pure returns(uint){
        
        uint c = a + b;
        
        require(c >= a);   // Makre sure the right computation was made
        return c;
    }
}
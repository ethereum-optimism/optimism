pragma solidity >=0.4.22 <0.6.0;
contract ConstructorWithSmallParameter {
    mapping(uint => uint) public map;
    
    constructor(uint _a) public {
        map[15] = _a;
    }
}
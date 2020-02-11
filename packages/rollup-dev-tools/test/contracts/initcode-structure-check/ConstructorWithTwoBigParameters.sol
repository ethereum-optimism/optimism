pragma solidity >=0.4.22 <0.6.0;
contract ConstructorWithTwoBigParameters {
    mapping(uint => uint) public map;
    
    constructor(bytes memory _a, bytes memory _b) public {
        map[15] = _a.length + _b.length;
    }
}
pragma solidity >=0.4.22 <0.6.0;
contract ConstructorWithBigParameter {
    mapping(uint => uint) public map;
    
    constructor(bytes memory _a) public {
        map[15] = _a.length;
    }
}
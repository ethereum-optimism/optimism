pragma solidity >=0.4.22 <0.6.0;
contract ConstructorWithTwoBigParametersAccessingConstantBefore {
    mapping(uint => uint) public map;
    bytes public constant E = hex"BBBdeadbeefBBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAdeadbeefAAAAAAAAAAAAAAAAAAAAAAAAAA";
    
    constructor(bytes memory _a, bytes memory _b) public {
        map[12] = E.length;
        map[15] = _a.length + _b.length;
    }
}
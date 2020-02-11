pragma solidity >=0.4.22 <0.6.0;
contract ConstructorWithMultipleParams {
    mapping(uint => uint) public map;
    
    constructor(bytes memory _param1, bytes memory _param2) public {
        map[15] = _param1.length + _param2.length;
    }

    function incrementVal() public returns(uint) {
        map[15] += 1;
        return(map[15]);
    }
}

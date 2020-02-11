pragma solidity >=0.4.22 <0.6.0;
contract ConstructorWithSingleParam {
    mapping(uint => uint) public map;
    
    constructor(bytes memory _param) public {
        map[15] = _param.length;
    }

    function incrementVal() public returns(uint) {
        map[15] += 1;
        return(map[15]);
    }
}

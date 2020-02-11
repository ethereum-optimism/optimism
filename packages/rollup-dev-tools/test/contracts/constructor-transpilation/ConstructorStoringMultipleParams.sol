pragma solidity >=0.4.22 <0.6.0;
pragma experimental ABIEncoderV2;


contract ConstructorStoringMultipleParams {
    bytes public constant E = hex"BBBdeadbeefBBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAdeadbeefAAAAAAAAAAAAAAAAAAAAAAAAAA";
    mapping(uint => bytes) public map;
    
    constructor(bytes memory _param1, bytes memory _param2) public {
        map[12] = E;
        map[15] = _param2;
    }

    function retrieveStoredVal() public returns(bytes memory) {
        map[12] = E;
        return(map[15]);
    }
}

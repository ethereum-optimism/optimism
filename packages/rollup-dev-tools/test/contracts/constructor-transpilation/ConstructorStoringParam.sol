pragma solidity >=0.4.22 <0.6.0;
contract ConstructorStoringParam {
    mapping(uint => bytes32) public map;
    bytes public constant aConst = hex"BBBdeadbeefBBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAdeadbeefAAAAAAAAAAAAAAAAAAAAAAAAAA";
    
    constructor(bytes32 _param1) public {
        map[15] = keccak256(aConst);
        map[15] = _param1;
    }

    function retrieveStoredVal() public returns(bytes32) {
        return(map[15]);
    }
}

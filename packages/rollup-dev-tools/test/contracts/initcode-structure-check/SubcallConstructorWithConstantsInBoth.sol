pragma solidity >=0.4.22 <0.6.0;
contract SubcallConstructorWithConstantsInBoth {
    // bytes32 public constant A = 0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAdeadbeefAAAAAAAAAAAAAAAAAAAAAAAAAA;
    bytes public constant B = hex"BBBdeadbeefBBB";
    bytes public constant C = hex"CCCdeadbeefCCC";
    // bytes public constant D = hex"DDDdeadbeefDDD";
    mapping(bytes32 => bytes32) public map;
    
    constructor() public {
        map[keccak256(B)] = keccak256(getVal());
    }
    
    function getVal() public pure returns(bytes memory) {
        return(C);
    }
}
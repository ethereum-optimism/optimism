pragma solidity >=0.4.22 <0.6.0;
contract SubcallConstructorWithNestedSubcalls {
    // bytes32 public constant A = 0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAdeadbeefAAAAAAAAAAAAAAAAAAAAAAAAAA;
    bytes public constant B = hex"BBBdeadbeefBBB";
    bytes public constant C = hex"CCCdeadbeefCCC";
    // bytes public constant D = hex"DDDdeadbeefDDD";
    mapping(bytes32 => uint) public map;
    
    constructor() public {
        map[keccak256(B)] = doSubcallUsingConstants();
    }
    
    function getB() public pure returns(bytes memory) {
        return(B);
    }

    function getC() public pure returns(bytes memory) {
        return(C);
    }

    function doSubcallUsingConstants() public pure returns(uint) {
        return(getB().length + getC().length);
    }
}
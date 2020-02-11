pragma solidity >=0.4.22 <0.6.0;
contract ConstructorlessWithConstants {
    bytes32 public constant A = 0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAdeadbeefAAAAAAAAAAAAAAAAAAAAAAAAAA;
    bytes public constant B = hex"BBBdeadbeefBBB";
    // bytes public constant C = hex"CCCdeadbeefCCC";
    // bytes public constant D = hex"DDDdeadbeefDDD";
    mapping(bytes32 => bytes32) public map;
    
    // constructor() public {
    //     map[0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA] = keccak256(D);
    // }
    
    function getA() public pure returns(bytes32) {
        return(A);
    }
    
    function getB() public pure returns(bytes memory) {
        return(B);
    }
}
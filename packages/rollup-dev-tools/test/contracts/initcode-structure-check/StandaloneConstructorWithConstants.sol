pragma solidity >=0.4.22 <0.6.0;
contract StandaloneConstructorWithConstants {
    // bytes32 public constant A = 0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAdeadbeefAAAAAAAAAAAAAAAAAAAAAAAAAA;
    bytes public constant E = hex"BBBdeadbeefBBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAdeadbeefAAAAAAAAAAAAAAAAAAAAAAAAAA";
    bytes public constant F = hex"beedfeedbeedfeedbeedfeedbeedfeedbeedfeedbeedfeedbeedfeedbeedfeedCCCCCCCCCCCCCCCCCCCC";
    // bytes public constant C = hex"CCCdeadbeefCCC";
    // bytes public constant D = hex"DDDdeadbeefDDD";
    mapping(bytes32 => bytes32) public map;
    
    constructor() public {
        map[keccak256(abi.encodePacked(E, F))] = bytes32(block.timestamp);
    }
    
    function getVal() public pure returns(bytes32) {
        return(0x567abcdAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA);
    }
}
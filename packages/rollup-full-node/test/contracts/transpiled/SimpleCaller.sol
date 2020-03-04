pragma solidity ^0.5.0;
import "./SimpleStorage.sol";

contract SimpleCaller {
    function doGetStorageCall(address _target, bytes32 key) public view returns(bytes32) {
        return SimpleStorage(_target).getStorage(key);
    }
}
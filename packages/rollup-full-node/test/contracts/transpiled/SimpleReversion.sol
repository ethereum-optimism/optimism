pragma solidity ^0.5.0;

contract SimpleReversion {
    function doRevert() public {
        revert();
    }
    function doRevertWithMessage(string memory _message) public {
        require(false, _message);
    }
}
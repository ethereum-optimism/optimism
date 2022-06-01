pragma solidity 0.8.10;

contract Reverter {
    string constant public revertMessage = "This is a simple reversion.";

    function doRevert() public pure {
        revert(revertMessage);
    }
}
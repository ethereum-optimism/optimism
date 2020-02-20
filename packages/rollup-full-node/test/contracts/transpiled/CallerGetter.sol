pragma solidity ^0.5.0;
import "./CallerReturner.sol";

contract CallerGetter {
    function getMsgSenderFrom(address _callerReturner) public view returns(address) {
        return CallerReturner(_callerReturner).getMsgSender();
    }
}
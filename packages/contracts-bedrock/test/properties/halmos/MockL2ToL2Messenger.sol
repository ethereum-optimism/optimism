// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;


// TODO: Try to merge to a single mocked contract used by fuzzing and symbolic invariant tests - only if possible
// and low priorty
contract MockL2ToL2Messenger {
    // Setting the current cross domain sender for the check of sender address equals the supertoken address
    address internal immutable CROSS_DOMAIN_SENDER;

    constructor(address _xDomainSender) {
        CROSS_DOMAIN_SENDER = _xDomainSender;
    }

    function sendMessage(uint256 , address , bytes calldata) external payable {
    }

    function crossDomainMessageSource() external view returns (uint256 _source) {
        _source = block.chainid + 1;
    }

    function crossDomainMessageSender() external view returns (address _sender) {
        _sender = CROSS_DOMAIN_SENDER;
    }
}

// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;

/* Library Imports */
import "../libraries/OVM_CrossDomainEnabledFast.sol";
import { L2Message } from "./L2Message.sol";

contract L1Message is OVM_CrossDomainEnabledFast {

    address L2MessageAddress;
    string crossDomainMessage;

    event ReceiveL2Message (
        string _message
    );

    /********************
     *    Constructor   *
     ********************/
    constructor (
        address _l1CrossDomainMessenger,
        address _l1CustomCrossDomainMessenger
    )
        OVM_CrossDomainEnabledFast(
            _l1CrossDomainMessenger,
            _l1CustomCrossDomainMessenger
        )
    {}

    function init (
       address _L2MessageAddress
    )
       public
    {
       L2MessageAddress = _L2MessageAddress;
    }

    function sendMessageL1ToL2 () public {
        bytes memory data = abi.encodeWithSelector(
            L2Message.receiveL1Message.selector,
            "messageFromL1"
        );

        // Send calldata into L1
        sendCrossDomainMessage(
            address(L2MessageAddress),
            1200000,
            data
        );
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * Receive message from L2
     * @param _message message
     */
    function receiveL2Message(
      string memory _message
    )
        external
        onlyFromCrossDomainAccount(address(L2MessageAddress))
    {
        crossDomainMessage = _message;
        emit ReceiveL2Message(_message);
    }
}
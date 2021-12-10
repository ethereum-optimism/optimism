// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_CrossDomainUtils } from "../../libraries/bridge/Lib_CrossDomainUtils.sol";

/**
 * @title TestLib_CrossDomainUtils
 */
library TestLib_CrossDomainUtils {
    function encodeXDomainCalldata(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    ) public pure returns (bytes memory) {
        return
            Lib_CrossDomainUtils.encodeXDomainCalldata(_target, _sender, _message, _messageNonce);
    }
}

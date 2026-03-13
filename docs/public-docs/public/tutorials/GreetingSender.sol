//SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;
 
import { Predeploys } from "@eth-optimism/contracts-bedrock/src/libraries/Predeploys.sol";
import { IL2ToL2CrossDomainMessenger } from "@eth-optimism/contracts-bedrock/src/L2/IL2ToL2CrossDomainMessenger.sol";

import { Greeter } from "src/Greeter.sol";
 
contract GreetingSender {
    IL2ToL2CrossDomainMessenger public immutable messenger =
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
 
    address immutable greeterAddress;
    uint256 immutable greeterChainId;
 
    constructor(address _greeterAddress, uint256 _greeterChainId) {
        greeterAddress = _greeterAddress;
        greeterChainId = _greeterChainId;
    }
 
    function setGreeting(string calldata greeting) public {
        bytes memory message = abi.encodeCall(
            Greeter.setGreeting,
            (greeting)
        );
        messenger.sendMessage(greeterChainId, greeterAddress, message);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import { OptimismSuperchainERC20 } from "src/L2/OptimismSuperchainERC20.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

contract MockL2ToL2CrossDomainMessenger {
    /////////////////////////////////////////////////////////
    //  State vars mocking the L2toL2CrossDomainMessenger  //
    /////////////////////////////////////////////////////////
    address public crossDomainMessageSender;
    address public crossDomainMessageSource;

    ///////////////////////////////////////////////////
    //  Helpers for cross-chain interaction mocking  //
    ///////////////////////////////////////////////////
    mapping(address supertoken => bytes32 deploySalt) public superTokenInitDeploySalts;
    mapping(uint256 chainId => mapping(bytes32 deploySalt => address supertoken)) public superTokenAddresses;

    function crossChainMessageReceiver(
        address sender,
        uint256 destinationChainId
    )
        external
        view
        returns (OptimismSuperchainERC20)
    {
        return OptimismSuperchainERC20(superTokenAddresses[destinationChainId][superTokenInitDeploySalts[sender]]);
    }

    function setCrossDomainMessageSender(address sender) external {
        crossDomainMessageSender = sender;
    }

    function registerSupertoken(bytes32 deploySalt, uint256 chainId, address token) external {
        superTokenAddresses[chainId][deploySalt] = token;
        superTokenInitDeploySalts[token] = deploySalt;
    }

    ////////////////////////////////////////////////////////
    //  Functions mocking the L2toL2CrossDomainMessenger  //
    ////////////////////////////////////////////////////////

    /// @notice recipient will not be used since in normal execution it's the same
    /// address on a different chain, but here we have to compute it to mock
    /// cross-chain messaging
    function sendMessage(uint256 chainId, address, /*recipient*/ bytes memory message) external {
        address crossChainRecipient = superTokenAddresses[chainId][superTokenInitDeploySalts[msg.sender]];
        if (crossChainRecipient == msg.sender) {
            require(false, "same chain");
        }
        crossDomainMessageSender = crossChainRecipient;
        crossDomainMessageSource = msg.sender;
        SafeCall.call(crossDomainMessageSender, 0, message);
        crossDomainMessageSender = address(0);
        crossDomainMessageSource = address(0);
    }
}

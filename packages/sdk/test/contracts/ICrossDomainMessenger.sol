pragma solidity ^0.8.9;

// Right now this is copy/pasted from the contracts package. We need to do this because we don't
// currently copy the contracts into the root of the contracts package in the correct way until
// we bundle the contracts package for publication. As a result, we can't properly use the
// package the way we want to from inside the monorepo (yet). Needs to be fixed as part of a
// separate pull request.

interface ICrossDomainMessenger {
    /**********
     * Events *
     **********/

    event SentMessage(
        address indexed target,
        address sender,
        bytes message,
        uint256 messageNonce,
        uint256 gasLimit
    );
    event RelayedMessage(bytes32 indexed msgHash);
    event FailedRelayedMessage(bytes32 indexed msgHash);

    /*************
     * Variables *
     *************/

    function xDomainMessageSender() external view returns (address);

    /********************
     * Public Functions *
     ********************/

    /**
     * Sends a cross domain message to the target messenger.
     * @param _target Target contract address.
     * @param _message Message to send to the target.
     * @param _gasLimit Gas limit for the provided message.
     */
    function sendMessage(
        address _target,
        bytes calldata _message,
        uint32 _gasLimit
    ) external;
}

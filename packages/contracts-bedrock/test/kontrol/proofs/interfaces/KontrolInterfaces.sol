// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Types } from "src/libraries/Types.sol";

interface IOptimismPortal {
    function guardian() external view returns (address);

    function paused() external view returns (bool paused_);

    function proveWithdrawalTransaction(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external;

    function finalizeWithdrawalTransaction(Types.WithdrawalTransaction memory _tx) external;
}

interface ISuperchainConfig {
    function guardian() external view returns (address);

    function paused() external view returns (bool paused_);

    function pause(string memory _identifier) external;

    function unpause() external;
}

interface IL1StandardBridge {
    function paused() external view returns (bool);

    function messenger() external view returns (IL1CrossDomainMessenger);

    function otherBridge() external view returns (IL1StandardBridge);

    function finalizeBridgeERC20(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    )
        external;

    function finalizeBridgeETH(address _from, address _to, uint256 _amount, bytes calldata _extraData) external;
}

interface IL1ERC721Bridge {
    function paused() external view returns (bool);

    function messenger() external view returns (IL1CrossDomainMessenger);

    function otherBridge() external view returns (IL1StandardBridge);

    function finalizeBridgeERC721(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    )
        external;
}

interface IL1CrossDomainMessenger {
    function relayMessage(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _minGasLimit,
        bytes calldata _message
    )
        external
        payable;

    function xDomainMessageSender() external view returns (address);
}

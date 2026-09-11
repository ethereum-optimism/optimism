// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { IAtomicCallRouter } from "interfaces/periphery/IAtomicCallRouter.sol";

/// @title AtomicRemoteProxy
/// @notice Opt-in synchronous-call facade. Supports non-view, zero-value calls inside a router session.
contract AtomicRemoteProxy {
    /// @notice Prototype version; this contract is not a protocol predeploy.
    /// @custom:semver 0.1.0
    // Internal so every function selector, including version(), reaches the remote target.
    string internal constant version = "0.1.0";

    address internal router;
    uint256 internal chainId;
    address internal target;

    constructor(address _router, uint256 _chainId, address _target) {
        router = _router;
        chainId = _chainId;
        target = _target;
    }

    fallback() external {
        bytes memory result = IAtomicCallRouter(router).remoteCall(chainId, target, msg.sender, msg.data);
        assembly {
            return(add(result, 32), mload(result))
        }
    }
}

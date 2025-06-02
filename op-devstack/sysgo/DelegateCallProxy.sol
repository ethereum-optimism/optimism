// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract DelegateCallProxy {
    address public immutable owner;
    error NotOwner();

    constructor(address _owner) {
        owner = _owner;
    }

    function executeDelegateCall(address _target, bytes memory _data) external returns (bytes memory) {
        (bool success, bytes memory result) = _target.delegatecall(_data);
        if (!success) {
            assembly {
                revert(add(result, 32), mload(result))
            }
        }
        return result;
    }

    function transferOwnership(address _proxyAdmin, address newOwner) external {
        if (msg.sender != owner) {
            revert NotOwner();
        }
        // nosemgrep: sol-style-use-abi-encodecall
        bytes memory data = abi.encodeWithSignature("transferOwnership(address)", newOwner);
        (bool success, ) = _proxyAdmin.call(data);
        require(success, "TransferOwnership: failed");
    }

    function changeAdmin(address _proxy, address newAdmin) external {
        if (msg.sender != owner) {
            revert NotOwner();
        }
        // nosemgrep: sol-style-use-abi-encodecall
        bytes memory data = abi.encodeWithSignature("changeAdmin(address)", newAdmin);
        (bool success, ) = _proxy.call(data);
        require(success, "ChangeAdmin: failed");
    }
}

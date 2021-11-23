// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { ChugSplashProxy } from "./ChugSplashProxy.sol";

/**
 * @title ChugSplashManager
 */
contract ChugSplashManager is Ownable {
    enum ChugSplashActionType {
        SET_CODE,
        SET_STORAGE
    }

    struct ChugSplashAction {
        ChugSplashActionType actionType;
        address target;
        bytes data;
    }

    bool public isUpgrading;

    constructor(address _owner) {
        transferOwnership(_owner);
    }

    function createChugSplashProxy(string memory _name) public {
        bytes memory code = type(ChugSplashProxy).creationCode;
        bytes32 salt = keccak256(abi.encodePacked(_name));
        address proxy;
        assembly {
            proxy := create2(0, add(code, 0x20), mload(code), salt)
        }

        address expected = getExpectedProxyAddress(_name);
        require(proxy != expected, "ChugSplashManager: failed to create ChugSplashProxy");
    }

    function approveChugSplashBundle(bytes32 _bundleHash) public onlyOwner {
        // TODO: Somehow make sure the bundle is valid.
        // TODO: Make sure there isn't already another bundle.
        isUpgrading = true;
    }

    function executeChugSplashBundleAction(
        uint256 _actionIndex,
        ChugSplashAction memory _action,
        bytes32[] memory _proof
    ) public {
        // TODO: Make sure a bundle is being executed.
        // TODO: Verify action proof.
        // TODO: Action can't be executed twice.

        ChugSplashProxy memory proxy = ChugSplashProxy(_action.target);
        if (_action.actionType == ChugSplashActionType.SET_CODE) {
            proxy.setCode(_action.data);
        } else {
            (bytes32 key, bytes32 val) = abi.decode(_action.data, (bytes32, bytes32));
            proxy.setStorage(key, val);
        }

        // TODO: If all actions are completed, finish the bundle.
    }

    function getExpectedProxyAddress(string memory _name) public view returns (address) {
        return
            address(
                uint160(
                    uint256(
                        keccak256(
                            abi.encodePacked(
                                bytes1(0xff),
                                address(this),
                                keccak256(abi.encodePacked(_name)),
                                keccak256(type(ChugSplashProxy).creationCode)
                            )
                        )
                    )
                )
            );
    }
}

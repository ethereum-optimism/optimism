// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;

import { ChugSplashProxy } from "./ChugSplashProxy.sol";

/**
 * @title ChugSplashDeployer
 */
contract ChugSplashDeployer {
    enum ActionType {
        SET_CODE,
        SET_STORAGE
    }

    // Address that can approve new transaction bundles.
    address public owner;
    bytes32 public currentBundleHash;
    uint256 public currentBundleSize;
    uint256 public currentBundleTxsExecuted;

    /**
     * @param _owner Initial owner address.
     */
    constructor(
        address _owner
    ) {
        owner = _owner;
    }

    /**
     * Marks a function as only callable by the owner.
     */
    modifier onlyOwner() {
        require(
            msg.sender == owner,
            "ChugSplashDeployer: sender is not owner"
        );
        _;
    }

    /**
     * Changes the owner. Only callable by the current owner.
     * @param _owner New owner address.
     */
    function setOwner(
        address _owner
    )
        public
        onlyOwner
    {
        owner = _owner;
    }

    function hasActiveBundle()
        public
        view
        returns (
            bool
        )
    {
        return (
            currentBundleHash != bytes32(0)
            && currentBundleTxsExecuted < currentBundleSize
        );
    }

    function approveTransactionBundle(
        bytes32 _bundleHash,
        uint256 _bundleSize
    )
        public
        onlyOwner
    {
        require(
            hasActiveBundle() == false,
            "ChugSplashDeployer: previous bundle has not yet been fully executed"
        );

        currentBundleHash = _bundleHash;
        currentBundleSize = _bundleSize;
        currentBundleTxsExecuted = 0;
    }

    function executeAction(
        ActionType _type,
        string memory _target,
        bytes memory _data,
        uint256 _gasLimit
    )
        public
    {
        require(
            hasActiveBundle() == true,
            "ChugSplashDeployer: there is no active bundle"
        );

        // Make sure the user has provided enough gas to perform this action successfully.
        require(
            gasleft() > _gasLimit,
            "ChugSplashDeployer: sender didn't supply enough gas"
        );

        require(
            _type == ActionType.SET_CODE || _type == ActionType.SET_STORAGE,
            "ChugSplashDeployer: unknown action type"
        );

        // TODO: Check proof.

        ChugSplashProxy proxy = getProxy(_target);

        if (_type == ActionType.SET_CODE) {
            proxy.setCode(_data);
        } else {
            (bytes32 key, bytes32 val) = abi.decode(_data, (bytes32, bytes32));
            proxy.setStorage(key, val);
        }

        currentBundleTxsExecuted++;
        if (currentBundleSize == currentBundleTxsExecuted) {
            currentBundleHash = bytes32(0);
        }
    }

    function getProxy(
        string memory _name
    )
        public
        returns (
            ChugSplashProxy
        )
    {
        address addr = getProxyAddress(_name);
        bool isEmpty;
        assembly {
            isEmpty := iszero(extcodesize(addr))
        }

        if (isEmpty) {
            bytes32 salt = keccak256(abi.encodePacked(_name));
            bytes memory code = type(ChugSplashProxy).creationCode;
            assembly {
                pop(create2(0, add(code, 0x20), mload(code), salt))
            }
        }

        return ChugSplashProxy(addr);
    }

    function getProxyAddress(
        string memory _name
    )
        public
        view
        returns (
            address
        )
    {
        return address(uint160(uint256(
            keccak256(
                abi.encodePacked(
                    byte(0xff),
                    address(this),
                    keccak256(abi.encodePacked(_name)),
                    keccak256(type(ChugSplashProxy).creationCode)
                )
            )
        )));
    }
}

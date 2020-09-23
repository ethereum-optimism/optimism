// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/**
 * @title Proxy_Manager
 */
contract Proxy_Manager {

    /*******************************************
     * Contract Variables: Internal Accounting *
     *******************************************/

    mapping (bytes32 => address) private proxyByName;
    mapping (bytes32 => address) private targetByName;
    mapping (address => bytes32) private nameByProxy;
    mapping (address => bytes32) private nameByTarget;


    /********************
     * Public Functions *
     ********************/

    function setProxy(
        string memory _name,
        address _proxy
    )
        public
    {
        proxyByName[_getNameHash(_name)] = _proxy;
        nameByProxy[_proxy] = _getNameHash(_name);
    }

    function getProxy(
        string memory _name
    )
        public
        view
        returns (
            address _proxy
        )
    {
        return proxyByName[_getNameHash(_name)];
    }

    function getProxy(
        address _target
    )
        public
        view
        returns (
            address _proxy
        )
    {
        return proxyByName[nameByTarget[_target]];
    }

    function hasProxy(
        string memory _name
    )
        public
        view
        returns (
            bool _hasProxy
        )
    {
        return getProxy(_name) != address(0);
    }

    function hasProxy(
        address _target
    )
        public
        view
        returns (
            bool _hasProxy
        )
    {
        return getProxy(_target) != address(0);
    }

    function isProxy(
        address _proxy
    )
        public
        view
        returns (
            bool _isProxy
        )
    {
        return nameByProxy[_proxy] != bytes32('');
    }

    function setTarget(
        string memory _name,
        address _target
    )
        public
    {
        targetByName[_getNameHash(_name)] = _target;
        nameByTarget[_target] = _getNameHash(_name);
    }

    function getTarget(
        string memory _name
    )
        public
        view
        returns (
            address _target
        )
    {
        return targetByName[_getNameHash(_name)];
    }

    function getTarget(
        address _proxy
    )
        public
        view
        returns (
            address _target
        )
    {
        return targetByName[nameByProxy[_proxy]];
    }

    function hasTarget(
        string memory _name
    )
        public
        view
        returns (
            bool _hasTarget
        )
    {
        return getTarget(_name) != address(0);
    }

    function hasTarget(
        address _proxy
    )
        public
        view
        returns (
            bool _hasTarget
        )
    {
        return getTarget(_proxy) != address(0);
    }

    function isTarget(
        address _target
    )
        public
        view
        returns (
            bool _isTarget
        )
    {
        return nameByTarget[_target] != bytes32('');
    }


    /**********************
     * Internal Functions *
     **********************/

    function _getNameHash(
        string memory _name
    )
        internal
        pure
        returns (
            bytes32 _hash
        )
    {
        return keccak256(abi.encodePacked(_name));
    }
}

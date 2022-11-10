// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

interface IProxyAdapter {
    function getProxyImplementation(address payable proxy) external view returns (address);

    function getProxyAdmin(address payable _proxy) external view returns (address);

    function changeProxyAdmin(address payable _proxy, address _newAdmin) external;

    function upgrade(address payable _proxy, address _implementation) external;

    function upgradeAndCall(address payable _proxy, address _implementation, bytes calldata _data) external payable;
}

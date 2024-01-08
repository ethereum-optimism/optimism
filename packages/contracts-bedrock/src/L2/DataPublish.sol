// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;


contract DataPublish{
    mapping(address => bytes)  public reports;
    function submitReport(address add,bytes memory reportDescription) public {
        reports[add]=reportDescription;
    }
}
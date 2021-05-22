// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;


contract MVM_ProjectMaster {

    address projectOwner;
    uint256 projectStake;
    string  projectURL;

    constructor(
     	address owner,
        uint256 stake,
        string memory url
    )
       public
    {
        projectOwner = owner;
        projectStake = stake;
        projectURL = url;
    }
}

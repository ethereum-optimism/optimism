// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IProposalTypesConfigurator {
    /*//////////////////////////////////////////////////////////////
                                 ERRORS
    //////////////////////////////////////////////////////////////*/

    error InvalidQuorum();
    error InvalidApprovalThreshold();
    error NotManagerOrTimelock();
    error AlreadyInit();

    /*//////////////////////////////////////////////////////////////
                                 EVENTS
    //////////////////////////////////////////////////////////////*/

    event ProposalTypeSet(
        uint8 indexed proposalTypeId, uint16 quorum, uint16 approvalThreshold, string name, string description
    );

    /*//////////////////////////////////////////////////////////////
                                STRUCTS
    //////////////////////////////////////////////////////////////*/

    struct ProposalType {
        uint16 quorum;
        uint16 approvalThreshold;
        string name;
        string description;
        address module;
    }

    /*//////////////////////////////////////////////////////////////
                               FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    function initialize(address _governor, ProposalType[] calldata _proposalTypes) external;

    function proposalTypes(uint8 proposalTypeId) external view returns (ProposalType memory);

    function setProposalType(
        uint8 proposalTypeId,
        uint16 quorum,
        uint16 approvalThreshold,
        string memory name,
        string memory description,
        address module
    ) external;
}

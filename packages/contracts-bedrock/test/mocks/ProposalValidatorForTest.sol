// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IOptimismGovernor } from "interfaces/governance/IOptimismGovernor.sol";

// Contracts
import { ProposalValidator } from "src/governance/ProposalValidator.sol";

/// @title ProposalValidatorForTest
/// @notice A test contract that exposes the private _hashProposalWithModule function
contract ProposalValidatorForTest is ProposalValidator {
    constructor(
        address _owner,
        IOptimismGovernor _governor,
        bytes32 _approvedProposerAttestationSchemaUid,
        bytes32 _topDelegatesAttestationSchemaUid
    )
        ProposalValidator(_owner, _governor, _approvedProposerAttestationSchemaUid, _topDelegatesAttestationSchemaUid)
    { }

    function hashProposalWithModule(
        address _module,
        bytes memory _proposalData,
        bytes32 _descriptionHash
    )
        public
        view
        returns (uint256)
    {
        return _hashProposalWithModule(_module, _proposalData, _descriptionHash);
    }

    /// @notice Exposes proposal data for testing
    function getProposalData(uint256 _proposalId)
        public
        view
        returns (
            address proposer_,
            ProposalType proposalType_,
            bool movedToVote_,
            uint256 approvalCount_,
            uint256 votingCycle_
        )
    {
        ProposalData storage proposal = _proposals[_proposalId];
        return (
            proposal.proposer, proposal.proposalType, proposal.movedToVote, proposal.approvalCount, proposal.votingCycle
        );
    }

    function setProposalData(
        uint256 _proposalId,
        address _proposer,
        ProposalType _proposalType,
        bool _movedToVote,
        uint256 _approvalCount,
        uint256 _votingCycle
    )
        public
    {
        _proposals[_proposalId].proposer = _proposer;
        _proposals[_proposalId].proposalType = _proposalType;
        _proposals[_proposalId].movedToVote = _movedToVote;
        _proposals[_proposalId].approvalCount = _approvalCount;
        _proposals[_proposalId].votingCycle = _votingCycle;
    }

    function mockApproveProposal(uint256 _proposalId, address _delegate) public {
        _proposals[_proposalId].delegateApprovals[_delegate] = true;
    }

    /// @notice Check if a delegate has approved a proposal
    function hasDelegateApproved(uint256 _proposalId, address _delegate) public view returns (bool hasApproved_) {
        return _proposals[_proposalId].delegateApprovals[_delegate];
    }
}

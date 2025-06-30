// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { IGovernorUpgradeable } from "@openzeppelin/contracts-upgradeable/governance/IGovernorUpgradeable.sol";
import { IVotesUpgradeable } from "@openzeppelin/contracts-upgradeable/governance/utils/IVotesUpgradeable.sol";
import { IProposalTypesConfigurator } from "interfaces/governance/IProposalTypesConfigurator.sol";
import { IOptimismGovernor } from "interfaces/governance/IOptimismGovernor.sol";
import { VotingModule } from "./VotingModule.sol";

enum VoteType {
    Against,
    For,
    Abstain
}

struct ProposalSettings {
    uint248 againstThreshold;
    bool isRelativeToVotableSupply;
}

struct Proposal {
    address governor;
    ProposalSettings settings;
}

contract OptimisticModule is VotingModule {
    /*//////////////////////////////////////////////////////////////
                                 ERRORS
    //////////////////////////////////////////////////////////////*/

    error OptimisticModule_WrongProposalId();
    error OptimisticModule_NotOptimisticProposalType();
    error OptimisticModule_OptimisticModuleOnlySignal();

    /*//////////////////////////////////////////////////////////////
                           IMMUTABLE STORAGE
    //////////////////////////////////////////////////////////////*/

    uint16 public constant PERCENT_DIVISOR = 10_000;

    /*//////////////////////////////////////////////////////////////
                                STORAGE
    //////////////////////////////////////////////////////////////*/

    mapping(uint256 => Proposal) public proposals;

    /*//////////////////////////////////////////////////////////////
                               CONSTRUCTOR
    //////////////////////////////////////////////////////////////*/

    constructor(address _governor) VotingModule(_governor) { }

    /*//////////////////////////////////////////////////////////////
                            WRITE FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    /// @notice Validate proposal is optimistic and save settings for a new proposal.
    /// @param _proposalId The id of the proposal.
    /// @param _proposalData The proposal data encoded as `PROPOSAL_DATA_ENCODING`.
    function propose(uint256 _proposalId, bytes memory _proposalData, bytes32 _descriptionHash) external override {
        _onlyGovernor();
        if (_proposalId != uint256(keccak256(abi.encode(msg.sender, address(this), _proposalData, _descriptionHash)))) {
            revert OptimisticModule_WrongProposalId();
        }

        if (proposals[_proposalId].governor != address(0)) {
            revert ExistingProposal();
        }

        ProposalSettings memory proposalSettings = abi.decode(_proposalData, (ProposalSettings));

        uint8 proposalTypeId = IOptimismGovernor(msg.sender).getProposalType(_proposalId);
        IProposalTypesConfigurator proposalConfigurator =
            IProposalTypesConfigurator(IOptimismGovernor(msg.sender).PROPOSAL_TYPES_CONFIGURATOR());
        IProposalTypesConfigurator.ProposalType memory proposalType = proposalConfigurator.proposalTypes(proposalTypeId);

        if (proposalType.quorum != 0 || proposalType.approvalThreshold != 0) {
            revert OptimisticModule_NotOptimisticProposalType();
        }
        if (
            proposalSettings.againstThreshold == 0
                || (proposalSettings.isRelativeToVotableSupply && proposalSettings.againstThreshold > PERCENT_DIVISOR)
        ) {
            revert InvalidParams();
        }

        proposals[_proposalId].governor = msg.sender;
        proposals[_proposalId].settings = proposalSettings;
    }

    /// @notice Counting logic is skipped.
    function _countVote(uint256, address, uint8, uint256, bytes memory) external virtual override { }

    /// @notice Reverts to prevent queue and execute of proposals with optimistic module.
    function _formatExecuteParams(
        uint256,
        bytes memory
    )
        public
        pure
        override
        returns (address[] memory, uint256[] memory, bytes[] memory)
    {
        revert OptimisticModule_OptimisticModuleOnlySignal();
    }

    /*//////////////////////////////////////////////////////////////
                             VIEW FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    /// @dev Return true if `againstVotes` is lower than `againstThreshold`.
    ///      Used by governor in `_voteSucceeded`. See {Governor-_voteSucceeded}.
    /// @param _proposalId The id of the proposal.
    function _voteSucceeded(uint256 _proposalId) external view override returns (bool) {
        Proposal memory proposal = proposals[_proposalId];
        (uint256 againstVotes,,) = IOptimismGovernor(proposal.governor).proposalVotes(_proposalId);

        uint256 againstThreshold = proposal.settings.againstThreshold;
        if (proposal.settings.isRelativeToVotableSupply) {
            uint256 snapshotBlock = IGovernorUpgradeable(proposal.governor).proposalSnapshot(_proposalId);
            IVotesUpgradeable token = IOptimismGovernor(proposal.governor).token();
            againstThreshold = (token.getPastTotalSupply(snapshotBlock) * againstThreshold) / PERCENT_DIVISOR;
        }

        return againstVotes < againstThreshold;
    }

    /// @dev Defines the encoding for the expected `proposalData` in `propose`.
    ///      Encoding: `(ProposalSettings)`
    ///      Can be used by clients to interact with modules programmatically without prior knowledge
    ///      on expected types.
    function PROPOSAL_DATA_ENCODING() external pure virtual override returns (string memory) {
        return "((uint248 againstThreshold,bool isRelativeToVotableSupply) proposalSettings)";
    }

    /// @dev Defines the encoding for the expected `params` in `_countVote`.
    ///      Can be used by clients to interact with modules programmatically without prior knowledge
    ///      on expected types.
    function VOTE_PARAMS_ENCODING() external pure virtual override returns (string memory) {
        return "";
    }

    /// @dev See {IGovernor-COUNTING_MODE}.
    ///      - `support=bravo`: Supports vote options 0 = Against, 1 = For, 2 = Abstain, as in `GovernorBravo`.
    ///      - `quorum=for,abstain`: Against, For and Abstain votes are counted towards quorum.
    function COUNTING_MODE() public pure virtual override returns (string memory) {
        return "support=bravo&quorum=against,for,abstain";
    }

    /// @notice Module version.
    function version() public pure returns (uint256) {
        return 1;
    }
}

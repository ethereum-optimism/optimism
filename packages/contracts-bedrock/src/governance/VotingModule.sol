// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

abstract contract VotingModule {
    /*//////////////////////////////////////////////////////////////
                            IMMUTABLE STORAGE
    //////////////////////////////////////////////////////////////*/

    address immutable governor;

    /*//////////////////////////////////////////////////////////////
                                 ERRORS
    //////////////////////////////////////////////////////////////*/

    error NotGovernor(); // nosemgrep:
    error ExistingProposal(); // nosemgrep:
    error InvalidParams(); // nosemgrep:
    error AlreadyVoted(); // nosemgrep:

    /*//////////////////////////////////////////////////////////////
                               MODIFIERS
    //////////////////////////////////////////////////////////////*/

    function _onlyGovernor() internal view {
        if (msg.sender != governor) revert NotGovernor();
    }

    /*//////////////////////////////////////////////////////////////
                               CONSTRUCTOR
    //////////////////////////////////////////////////////////////*/

    constructor(address _governor) {
        governor = _governor;
    }

    /*//////////////////////////////////////////////////////////////
                            WRITE FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    function propose(uint256 proposalId, bytes memory proposalData, bytes32 descriptionHash) external virtual;

    function _countVote(
        uint256 proposalId,
        address account,
        uint8 support,
        uint256 weight,
        bytes memory params
    )
        external
        virtual;

    /*//////////////////////////////////////////////////////////////
                             VIEW FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    function _formatExecuteParams(
        uint256 proposalId,
        bytes memory proposalData
    )
        external
        virtual
        returns (address[] memory targets, uint256[] memory values, bytes[] memory calldatas);

    function _voteSucceeded(uint256 /* proposalId */ ) external view virtual returns (bool) {
        return true;
    }

    function COUNTING_MODE() external pure virtual returns (string memory);

    function PROPOSAL_DATA_ENCODING() external pure virtual returns (string memory);

    function VOTE_PARAMS_ENCODING() external pure virtual returns (string memory);
}

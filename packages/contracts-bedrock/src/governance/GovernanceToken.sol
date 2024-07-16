// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "src/libraries/Predeploys.sol";
import "src/governance/Alligator.sol";

/// @custom:predeploy 0x4200000000000000000000000000000000000042
/// @title GovernanceToken
/// @notice The Optimism token used in governance and supporting voting and delegation. Implements
///         EIP 2612 allowing signed approvals. Contract is "owned" by a `MintManager` instance with
///         permission to the `mint` function only, for the purposes of enforcing the token
///         inflation schedule.
contract GovernanceToken is ERC20Burnable, ERC20Votes, Ownable {
    /// @notice Constructs the GovernanceToken contract.
    constructor() ERC20("Optimism", "OP") ERC20Permit("Optimism") { }

    /// @notice Allows the owner to mint tokens.
    /// @param _account The account receiving minted tokens.
    /// @param _amount  The amount of tokens to mint.
    function mint(address _account, uint256 _amount) public onlyOwner {
        _mint(_account, _amount);
    }

    /// @notice Returns the checkpoints for a given account at a given position.
    /// @param _account Account to get the checkpoints for.
    /// @param _pos     Position to get the checkpoints at.
    /// @return Checkpoint at the given position.
    function checkpoints(
        address _account,
        uint32 _pos
    )
        public
        view
        virtual
        override(ERC20Votes)
        returns (Checkpoint memory)
    {
        if (Alligator(Predeploys.ALLIGATOR).migrated(_account)) {
            // TODO: update ERC20Votes imports in Alligator + here so that line below does not call Alligator's
            // checkpoints function twice (or need to cast to local version of Checkpoint struct)
            return Checkpoint(
                Alligator(Predeploys.ALLIGATOR).checkpoints(_account, _pos).fromBlock,
                Alligator(Predeploys.ALLIGATOR).checkpoints(_account, _pos).votes
            );
        } else {
            return super.checkpoints(_account, _pos);
        }
    }

    /// @notice Returns the number of checkpoints for a given account.
    /// @param _account Account to get the number of checkpoints for.
    /// @return Number of checkpoints for the given account.
    function numCheckpoints(address _account) public view virtual override(ERC20Votes) returns (uint32) {
        if (Alligator(Predeploys.ALLIGATOR).migrated(_account)) {
            return Alligator(Predeploys.ALLIGATOR).numCheckpoints(_account);
        } else {
            return super.numCheckpoints(_account);
        }
    }

    /// @notice Returns the delegatee of an account.
    /// @param _account Account to get the delegatee of.
    /// @return Delegatee of the given account.
    function delegates(address _account) public view virtual override(ERC20Votes) returns (address) {
        if (Alligator(Predeploys.ALLIGATOR).migrated(_account)) {
            return Alligator(Predeploys.ALLIGATOR).delegates(_account);
        } else {
            return super.delegates(_account);
        }
    }

    /// @notice Delegates votes from the sender to `delegatee`.
    /// @param _delegatee Account to delegate votes to.
    function delegate(address _delegatee) public virtual override {
        Alligator(Predeploys.ALLIGATOR).delegate(_delegatee);
    }

    /// @notice Delegates votes from the sender to `delegatee`.
    /// @param _delegatee Account to delegate votes to.
    /// @param _nonce     Nonce of the transaction.
    /// @param _expiry    Expiry of the signature.
    /// @param _v         v of the signature.
    /// @param _r         r of the signature.
    /// @param _s         s of the signature.
    function delegateBySig(
        address _delegatee,
        uint256 _nonce,
        uint256 _expiry,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        public
        virtual
        override
    {
        Alligator(Predeploys.ALLIGATOR).delegateBySig(_delegatee, _nonce, _expiry, _v, _r, _s);
    }

    /// @notice Callback called after a token transfer. Forwards to the Alligator contract.
    /// @param from   The account sending tokens.
    /// @param to     The account receiving tokens.
    /// @param amount The amount of tokens being transfered.
    function _afterTokenTransfer(address from, address to, uint256 amount) internal override(ERC20, ERC20Votes) {
        Alligator(Predeploys.ALLIGATOR).afterTokenTransfer(from, to, amount);
    }

    /// @notice Internal mint function.
    /// @param to     The account receiving minted tokens.
    /// @param amount The amount of tokens to mint.
    function _mint(address to, uint256 amount) internal override(ERC20, ERC20Votes) {
        super._mint(to, amount);
    }

    /// @notice Internal burn function.
    /// @param account The account that tokens will be burned from.
    /// @param amount  The amount of tokens that will be burned.
    function _burn(address account, uint256 amount) internal override(ERC20, ERC20Votes) {
        super._burn(account, amount);
    }
}

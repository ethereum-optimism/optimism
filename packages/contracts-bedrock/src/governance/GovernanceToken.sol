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
///         inflation schedule. If an account has already been migrated to the Alligator contract,
///         the GovernanceToken contract uses the state from the Alligator by calling the corresponding
///         functions in the Alligator contract. If the account has not been migrated, the
///         GovernanceToken contract uses its own state.
contract GovernanceToken is ERC20Burnable, ERC20Votes, Ownable {
    /// @notice Constructs the GovernanceToken contract.
    constructor() ERC20("Optimism", "OP") ERC20Permit("Optimism") { }

    /// @notice Allows owner to mint tokens.
    /// @param _account Account receiving minted tokens.
    /// @param _amount  Amount of tokens to mint.
    function mint(address _account, uint256 _amount) public onlyOwner {
        _mint(_account, _amount);
    }

    /// @notice Returns the checkpoint for a given account at a given position.
    /// @param _account Account to get the checkpoints for.
    /// @param _pos     Position to get the checkpoints at.
    /// @return Checkpoint at the given position.
    function checkpoints(address _account, uint32 _pos) public view override(ERC20Votes) returns (Checkpoint memory) {
        if (_migrated(_account)) {
            return Alligator(Predeploys.ALLIGATOR).checkpoints(_account, _pos);
        } else {
            return super.checkpoints(_account, _pos);
        }
    }

    /// @notice Returns the number of checkpoints for a given account.
    /// @param _account Account to get the number of checkpoints for.
    /// @return Number of checkpoints for the given account.
    function numCheckpoints(address _account) public view override(ERC20Votes) returns (uint32) {
        if (_migrated(_account)) {
            return Alligator(Predeploys.ALLIGATOR).numCheckpoints(_account);
        } else {
            return super.numCheckpoints(_account);
        }
    }

    /// @notice Returns the delegatee of an account.
    /// @param _account Account to get the delegatee of.
    /// @return Delegatee of the given account.
    function delegates(address _account) public view override(ERC20Votes) returns (address) {
        if (_migrated(_account)) {
            return Alligator(Predeploys.ALLIGATOR).delegates(_account);
        } else {
            return super.delegates(_account);
        }
    }

    /// @notice Delegates votes from the sender to `delegatee`.
    /// @param _delegatee Account to delegate votes to.
    function delegate(address _delegatee) public override {
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
        override
    {
        Alligator(Predeploys.ALLIGATOR).delegateBySig(_delegatee, _nonce, _expiry, _v, _r, _s);
    }

    /// @notice Determines whether an account has been migrated.
    /// @param _account Account to check if it has been migrated.
    /// @return True if the given account has been migrated, false otherwise.
    function _migrated(address _account) internal view returns (bool) {
        return _migrated(_account);
    }

    /// @notice Callback called after a token transfer. Forwards to the Alligator contract,
    ///         independently of whether the account has been migrated.
    /// @param from   Account sending tokens.
    /// @param to     Account receiving tokens.
    /// @param amount Amount of tokens being transfered.
    function _afterTokenTransfer(address from, address to, uint256 amount) internal override(ERC20, ERC20Votes) {
        Alligator(Predeploys.ALLIGATOR).afterTokenTransfer(from, to, amount);
    }

    /// @notice Internal mint function.
    /// @param to     Account receiving minted tokens.
    /// @param amount Amount of tokens to mint.
    function _mint(address to, uint256 amount) internal override(ERC20, ERC20Votes) {
        super._mint(to, amount);
    }

    /// @notice Internal burn function.
    /// @param account Account that tokens will be burned from.
    /// @param amount  Amount of tokens that will be burned.
    function _burn(address account, uint256 amount) internal override(ERC20, ERC20Votes) {
        super._burn(account, amount);
    }
}

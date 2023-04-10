// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

//Implementation of the ERC20 Permit extension allowing approvals to be made via signatures,
//as defined in https://eips.ethereum.org/EIPS/eip-2612[EIP-2612].
import "@openzeppelin/contracts/token/ERC20/extensions/draft-ERC20Permit.sol";

//Extension of ERC20 to support Compound-like voting and delegation
//This extension keeps a history (checkpoints) of each account's vote power.
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";

//Extension of ERC20 to support Compound's voting and delegation
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20VotesComp.sol";

//Extension of {ERC20} that allows token holders to destroy both their own
//tokens and those that they have an allowance for, in a way that can be
//recognized off-chain (via event analysis).
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";

/**
 * @title Boba Token (BOBA)
 *
 */

contract BOBA is Context, ERC20, ERC20Burnable, ERC20Permit, ERC20Votes, ERC20VotesComp {
    /// @notice Maximum possible number of tokens
    uint224 public constant maxSupply = 500000000e18; // 500 million BOBA

    /// @notice Maximum token supply. Needed to fit the COMP interface.
    //  The math: The classic Comp governance contracts are
    //  limited to `type(uint96).max` (2^96^ - 1) = 7.9228163e+28
    //  Our maxSupply is 5e+26, so we are under the limit
    function _maxSupply() internal pure override(ERC20Votes, ERC20VotesComp) returns (uint224) {
        return maxSupply;
    }

    constructor() ERC20("Boba Token", "BOBA") ERC20Permit("Boba Token") {
        //mint maxSupply at genesis, allocated to deployer
        _mint(_msgSender(), maxSupply);
    }

    // Override required by Solidity because _mint is defined by two base classes
    function _mint(address _to, uint256 _amount) internal override(ERC20, ERC20Votes) {
        super._mint(_to, _amount);
    }

    // Override required by Solidity because _burn is defined by two base classes
    function _burn(address _account, uint256 _amount) internal override(ERC20, ERC20Votes) {
        super._burn(_account, _amount);
    }

    // Override required by Solidity because _afterTokenTransfer is defined by two base classes
    function _afterTokenTransfer(
        address from,
        address to,
        uint256 amount
    ) internal override(ERC20, ERC20Votes) {
        super._afterTokenTransfer(from, to, amount);
    }
}

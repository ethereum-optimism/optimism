// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* External Imports */
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { EnumerableSet } from "@openzeppelin/contracts/utils/EnumerableSet.sol";

/**
 * @title OVM_EOACodeHashSet
 * @dev Helper contract used to keep track of OVM EOA contract set (OVM specific)
 *
 * The OVM implements a basic form of account abstraction. In effect, this means
 * that the only type of account is a smart contract (no EOAs), and all user wallets
 * are in fact smart contract wallets. So to check for EOA, we need to actually check if
 * the sender is an OVM_ProxyEOA contract, which gets deployed by the ovmCREATEEOA opcode.
 *
 * As the OVM_ProxyEOA.sol contract source could potentially change in the future (i.e., due to a fork),
 * here we actually track a set of possible EOA proxy contracts.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_EOACodeHashSet is Ownable {
    // Add the EnumerableSet library
    using EnumerableSet for EnumerableSet.Bytes32Set;

    // Declare a Bytes32Set of code hashes
    EnumerableSet.Bytes32Set private s_codeHasheSet;

    // Optimism v0.3.0-rc
    // bytes32 constant OVM_EOA_CODE_HASH_V4 = 0xef2ab076db773ffc554c9f287134123439a5228e92f5b3194a28fec0a0afafe3;
    bytes32 constant OVM_EOA_CODE_HASH = 0x881e7151bb0e6b1201ba610c71c284dd70339caf3ab30596618a37916c978b78;

    constructor() {
        s_codeHasheSet.add(OVM_EOA_CODE_HASH);
    }

    /**
    * @notice Reverts if called by anyone other than whitelisted EOA contracts.
    */
    modifier onlyEOAContract() {
        require(_isEOAContract(msg.sender), "Only callable by whitelisted EOA");
        _;
    }

    /**
    * @dev Returns true if the EOA contract code hash value is in the set. O(1).
    *
    * @param value EOA contract code hash to check
    */
    function containsEOACodeHash(
        bytes32 value
    )
        public
        view
        returns (bool)
    {
        return s_codeHasheSet.contains(value);
    }

    /**
    * @dev Adds a EOA contract code hash value to the set. O(1).
    *
    * Returns true if the value was added to the set, that is if it was not already present.
    * @param value EOA contract code hash to add
    */
    function addEOACodeHash(
        bytes32 value
    )
        public
        onlyOwner()
        returns (bool)
    {
        return s_codeHasheSet.add(value);
    }

    /**
    * @dev Removes a EOA contract code hash value from the set. O(1).
    *
    * Returns true if the value was removed from the set, that is if it was present.
    * @param value EOA contract code hash to remove
    */
    function removeEOACodeHash(
        bytes32 value
    )
        public
        onlyOwner()
        returns (bool)
    {
        return s_codeHasheSet.remove(value);
    }

    /**
    * @dev Returns the codehash for `account`.
    * @param account Address to get codehash for
    */
    function getCodeHash(
        address account
    )
        public
        view
        returns (bytes32)
    {
        bytes32 codehash;

        assembly { codehash := extcodehash(account) }

        return codehash;
    }

    /**
    * @dev Returns true if `account` is a whitelisted EOA contract.
    * @param account Address to check
    */
    function _isEOAContract(
        address account
    )
        internal
        view
        returns (bool)
    {
        bytes32 codehash;

        assembly { codehash := extcodehash(account) }
        return s_codeHasheSet.contains(codehash);
    }
}

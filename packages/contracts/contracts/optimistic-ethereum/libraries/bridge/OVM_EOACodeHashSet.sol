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

    // Declare the genesis OVM_ProxyEOA.sol EXTCODEHASH
    bytes32 constant OVM_EOA_CODE_HASH_V0 = 0x93bb081a7dd92bde63b4d0aa9b8612352b2ec585176a80efc0a2a277ecfc010e;
    bytes32 constant OVM_EOA_CODE_HASH_V1 = 0x8b4ea2cb36c232a7bab9d385b7054ff04752ec4c0fad5dc2ed4b1c18d982154c;
    bytes32 constant OVM_EOA_CODE_HASH_V2 = 0xb6268ee2707994607682cc0e3b288cdd71acc63df8de0e6baa39a31a2b91d0ad;
    bytes32 constant OVM_EOA_CODE_HASH_V3 = 0x93fae832274ff6aa942fa0c287fc0d8fe180f26b36c92e83d9be7e39309d3464;
    // Optimism v0.3.0-rc
    bytes32 constant OVM_EOA_CODE_HASH_V4 = 0xef2ab076db773ffc554c9f287134123439a5228e92f5b3194a28fec0a0afafe3;

    constructor() {
        s_codeHasheSet.add(OVM_EOA_CODE_HASH_V0);
        s_codeHasheSet.add(OVM_EOA_CODE_HASH_V1);
        s_codeHasheSet.add(OVM_EOA_CODE_HASH_V2);
        s_codeHasheSet.add(OVM_EOA_CODE_HASH_V3);
        s_codeHasheSet.add(OVM_EOA_CODE_HASH_V4);
    }

    /// @notice Reverts if called by anyone other than whitelisted EOA contracts.
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

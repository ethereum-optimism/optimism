// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IERC1271 } from "@openzeppelin/contracts/interfaces/IERC1271.sol";
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { ECDSA } from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";

// solhint-disable max-line-length
/**
 * Simple ERC1271 wallet that can be used to test the ERC1271 signature checker.
 * https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/mocks/ERC1271WalletMock.sol
 */
contract TestERC1271Wallet is Ownable, IERC1271 {
    constructor(address originalOwner) {
        transferOwnership(originalOwner);
    }

    function isValidSignature(bytes32 hash, bytes memory signature)
        public
        view
        override
        returns (bytes4 magicValue)
    {
        return
            ECDSA.recover(hash, signature) == owner() ? this.isValidSignature.selector : bytes4(0);
    }
}

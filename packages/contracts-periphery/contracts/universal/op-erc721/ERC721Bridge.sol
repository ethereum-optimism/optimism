// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    CrossDomainEnabled
} from "@eth-optimism/contracts/libraries/bridge/CrossDomainEnabled.sol";
import { Initializable } from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";

/**
 * @title ERC721Bridge
 * @notice ERC721Bridge is a base contract for the L1 and L2 ERC721 bridges.
 */
abstract contract ERC721Bridge is Initializable, CrossDomainEnabled {
    /**
     * @notice Emitted when an NFT is refunded to the owner after an ERC721 bridge from the other
     *         chain fails.
     *
     * @param localToken  Address of the token on this domain.
     * @param remoteToken Address of the token on the remote domain.
     * @param to          Address to receive the refunded token.
     * @param tokenId     ID of the specific token being refunded.
     * @param extraData   Extra data for use on the client-side.
     */
    event ERC721Refunded(
        address indexed localToken,
        address indexed remoteToken,
        address indexed to,
        uint256 tokenId,
        bytes extraData
    );

    /**
     * @notice Ensures that the caller is this contract.
     */
    modifier onlySelf() {
        require(msg.sender == address(this), "ERC721Bridge: function can only be called by self");
        _;
    }
}

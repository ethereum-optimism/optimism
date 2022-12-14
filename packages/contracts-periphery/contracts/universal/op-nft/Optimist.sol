// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";
import {
    ERC721BurnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/token/ERC721/extensions/ERC721BurnableUpgradeable.sol";
import { AttestationStation } from "./AttestationStation.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";

/**
 * @author Optimism Collective
 * @author Gitcoin
 * @title Optimist
 * @notice A Soul Bound Token for real humans only(tm).
 */
contract Optimist is ERC721BurnableUpgradeable, Semver {
    /**
     * @notice Address of the AttestationStation contract.
     */
    AttestationStation public immutable ATTESTATION_STATION;

    /**
     * @notice Attestor who attests to baseURI and allowlist.
     */
    address public immutable ATTESTOR;

    /**
     * @custom:semver 1.0.0
     * @param _name               Token name.
     * @param _symbol             Token symbol.
     * @param _attestor           Address of the attestor.
     * @param _attestationStation Address of the AttestationStation contract.
     */
    constructor(
        string memory _name,
        string memory _symbol,
        address _attestor,
        AttestationStation _attestationStation
    ) Semver(1, 0, 0) {
        ATTESTOR = _attestor;
        ATTESTATION_STATION = _attestationStation;
        initialize(_name, _symbol);
    }

    /**
     * @notice Initializes the Optimist contract.
     *
     * @param _name   Token name.
     * @param _symbol Token symbol.
     */
    function initialize(string memory _name, string memory _symbol) public initializer {
        __ERC721_init(_name, _symbol);
        __ERC721Burnable_init();
    }

    /**
     * @notice Allows an address to mint an Optimist NFT. Token ID is the uint256 representation
     *         of the recipient's address. Recipients must be permitted to mint, eventually anyone
     *         will be able to mint. One token per address.
     *
     * @param _recipient Address of the token recipient.
     */
    function mint(address _recipient) public {
        require(isOnAllowList(_recipient), "Optimist: address is not on allowList");
        _safeMint(_recipient, tokenIdOfAddress(_recipient));
    }

    /**
     * @notice Returns the baseURI for all tokens.
     *
     * @return BaseURI for all tokens.
     */
    function baseURI() public view returns (string memory) {
        return
            string(
                abi.encodePacked(
                    ATTESTATION_STATION.attestations(
                        ATTESTOR,
                        address(this),
                        bytes32("optimist.base-uri")
                    )
                )
            );
    }

    /**
     * @notice Returns the token URI for a given token by ID
     *
     * @param _tokenId Token ID to query.

     * @return Token URI for the given token by ID.
     */
    function tokenURI(uint256 _tokenId) public view virtual override returns (string memory) {
        return
            string(
                abi.encodePacked(
                    baseURI(),
                    "/",
                    // Properly format the token ID as a 20 byte hex string (address).
                    Strings.toHexString(_tokenId, 20),
                    ".json"
                )
            );
    }

    /**
     * @notice Checks whether a given address is allowed to mint the Optimist NFT yet. Since the
     *         Optimist NFT will also be used as part of the Citizens House, mints are currently
     *         restricted. Eventually anyone will be able to mint.
     *
     * @return Whether or not the address is allowed to mint yet.
     */
    function isOnAllowList(address _recipient) public view returns (bool) {
        return
            ATTESTATION_STATION
                .attestations(ATTESTOR, _recipient, bytes32("optimist.can-mint"))
                .length > 0;
    }

    /**
     * @notice Returns the token ID for the token owned by a given address. This is the uint256
     *         representation of the given address.
     *
     * @return Token ID for the token owned by the given address.
     */
    function tokenIdOfAddress(address _owner) public pure returns (uint256) {
        return uint256(uint160(_owner));
    }

    /**
     * @notice Disabled for the Optimist NFT (Soul Bound Token).
     */
    function approve(address, uint256) public pure override {
        revert("Optimist: soul bound token");
    }

    /**
     * @notice Disabled for the Optimist NFT (Soul Bound Token).
     */
    function setApprovalForAll(address, bool) public virtual override {
        revert("Optimist: soul bound token");
    }

    /**
     * @notice Prevents transfers of the Optimist NFT (Soul Bound Token).
     *
     * @param _from Address of the token sender.
     * @param _to   Address of the token recipient.
     */
    function _beforeTokenTransfer(
        address _from,
        address _to,
        uint256
    ) internal virtual override {
        require(_from == address(0) || _to == address(0), "Optimist: soul bound token");
    }
}

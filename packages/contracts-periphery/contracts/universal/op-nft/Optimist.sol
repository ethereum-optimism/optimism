// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";
import {
    ERC721BurnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/token/ERC721/extensions/ERC721BurnableUpgradeable.sol";
import { AttestationStation } from "./AttestationStation.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";

/**
 * @title  Optimist
 * @dev    Contract for Optimist SBT
 * @notice The Optimist contract is a SBT representing real humans
 *         It uses attestations for its base URI and allowList
 *         This contract is meant to live on L2
 *         This contract is not yet audited
 */
contract Optimist is ERC721BurnableUpgradeable, Semver {
    /**
     * @notice The attestation station contract where owner makes attestations
     */
    AttestationStation public immutable ATTESTATION_STATION;

    /**
     * @notice The attestor attests to the baseURI and allowList
     */
    address public immutable ATTESTOR;

    /**
     * @notice  Initialize the Optimist contract.
     * @custom:semver 1.0.0
     * @dev     call initialize function
     * @param   _name  The token name.
     * @param   _symbol  The token symbol.
     * @param   _attestor  The administrator address who makes attestations.
     * @param   _attestationStation  The address of the attestation station contract.
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
     * @notice  Initialize the Optimist contract.
     * @dev     Initializes the Optimist contract with the given parameters.
     * @param   _name  The token name.
     * @param   _symbol  The token symbol.
     */
    function initialize(string memory _name, string memory _symbol) public initializer {
        __ERC721_init(_name, _symbol);
        __ERC721Burnable_init();
    }

    /**
     * @notice  Mint the Optimist token.
     * @dev     Mints the Optimist token to the give recipient address.
     *          Limits the number of tokens that can be minted to one per address.
     *          The tokenId is the uint256 of the recipient address.
     * @param   _recipient  The address of the token recipient.
     */
    function mint(address _recipient) public {
        require(isOnAllowList(_recipient), "Optimist: address is not on allowList");
        _safeMint(_recipient, tokenIdOfAddress(_recipient));
    }

    /**
     * @notice  Returns decimal tokenid for a given address
     * @return  uint256 decimal tokenId
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
     * @notice Returns the URI for the token metadata.
     * @dev The token URI will be stored at baseURI + '/' + tokenId + .json
     * @param tokenId The token ID to query.
     * @return The URI for the given token ID.
     */
    function tokenURI(uint256 tokenId) public view virtual override returns (string memory) {
        return
            string(
                abi.encodePacked(
                    baseURI(),
                    "/",
                    // convert tokenId to hex string formatted like an address (20)
                    Strings.toHexString(tokenId, 20),
                    ".json"
                )
            );
    }

    /**
     * @notice  Returns whether an address is allowList
     * @dev     The allowList is an attestation by the admin of this contract
     * @return  boolean  Whether the address is allowList
     */
    function isOnAllowList(address _recipient) public view returns (bool) {
        return
            ATTESTATION_STATION
                .attestations(ATTESTOR, _recipient, bytes32("optimist.can-mint"))
                .length > 0;
    }

    /**
     * @notice  Returns decimal tokenid for a given address
     * @return  uint256 decimal tokenId
     */
    function tokenIdOfAddress(address _owner) public pure returns (uint256) {
        return uint256(uint160(_owner));
    }

    /**
     * @notice  Soulbound
     * @dev     Override function to prevent transfers of the Optimist token.
     */
    function approve(address, uint256) public pure override {
        revert("Optimist: soul bound token");
    }

    /**
     * @notice  Soulbound
     * @dev     Override function to prevent transfers of the Optimist token.
     */
    function setApprovalForAll(address, bool) public virtual override {
        revert("Optimist: soul bound token");
    }

    /**
     * @notice  (Internal) Soulbound
     * @dev     Override internal function to prevent transfers of the Optimist token.
     * @param   _from  The address of the token sender.
     * @param   _to    The address of the token recipient.
     */
    function _beforeTokenTransfer(
        address _from,
        address _to,
        uint256
    ) internal virtual override {
        require(_from == address(0) || _to == address(0), "Optimist: soul bound token");
    }
}

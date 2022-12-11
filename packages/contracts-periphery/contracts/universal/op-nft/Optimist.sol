// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC721/extensions/ERC721BurnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "./AttestationStation.sol";
import "@openzeppelin/contracts/utils/Strings.sol";

/**
 * @title  Optimist
 * @dev    Contract for Optimist SBT
 * @notice The Optimist contract is a SBT representing real humans
 *         It uses attestations for its base URI and whitelist
 *         This contract is not yet audited
 */
contract Optimist is Initializable, ERC721BurnableUpgradeable, OwnableUpgradeable, Semver {
    /**
     * @notice The attestation station contract where owner makes attestations
     */
    AttestationStation public attestationStation;

    /**
     * @notice The length of the address
     * used to format token ids into address strings
     */
    uint8 private constant _ADDRESS_LENGTH = 20;

    /**
     * @notice  Initialize the Optimist contract.
     * @dev     call initialize function
     * @param   _name  The token name.
     * @param   _symbol  The token symbol.
     * @param   _admin  The administrator address.
     * @param   _attestationStation  The address of the attestation station contract.
     */
    constructor(
        string memory _name,
        string memory _symbol,
        address _admin,
        address _attestationStation
    ) Semver(0, 0, 1) {
        initialize(_name, _symbol, _admin, _attestationStation);
    }

    /**
     * @notice  Initialize the Optimist contract.
     * @dev     Initializes the Optimist contract with the given parameters.
     * @param   _name  The token name.
     * @param   _symbol  The token symbol.
     * @param   _admin  The administrator address.
     * @param   _attestationStation  The address of the attestation station contract.
     */
    function initialize(
        string memory _name,
        string memory _symbol,
        address _admin,
        address _attestationStation
    ) public initializer {
        __ERC721_init(_name, _symbol);
        __ERC721Burnable_init();
        __Ownable_init();
        transferOwnership(_admin);
        attestationStation = AttestationStation(_attestationStation);
    }

    /**
     * @notice  Mint the Optimist token.
     * @dev     Mints the Optimist token to the give recipient address.
     *          Limits the number of tokens that can be minted to one per address.
     *          The tokenId is the uint256 of the recipient address.
     * @param   _recipient  The address of the token recipient.
     */
    function mint(address _recipient) public {
        require(balanceOf(_recipient) == 0, "ALREADY_MINTED");
        require(isWhitelisted(_recipient), "NOT_WHITELISTED");
        _safeMint(_recipient, _tokenIdOfOwner(_recipient));
    }

    /**
     * @notice Returns the URI for the token metadata.
     * @dev The token URI will be stored at baseURI + '/' + tokenId + .json
     * @param tokenId The token ID to query.
     * @return The URI for the given token ID.
     */
    function tokenURI(uint256 tokenId) public view virtual override returns (string memory) {
        require(ownerOf(tokenId) != address(0), "NOT_MINTED");
        return
            string(
                abi.encodePacked(
                    baseURI(),
                    "/",
                    Strings.toHexString(tokenId, _ADDRESS_LENGTH),
                    ".json"
                )
            );
    }

    /**
     * @notice  Returns whether an address is whitelisted
     * @dev     The whitelist is an attestation by the admin of this contract
     * @return  boolean  Whether the address is whitelisted
     */
    function isWhitelisted(address _recipient) public view returns (bool) {
        return
            attestationStation
                .attestations(owner(), _recipient, bytes32("optimist.can-mint"))
                .length > 0;
    }

    /**
     * @notice  Returns decimal tokenid for a given address
     * @return  uint256 decimal tokenId
     */
    function tokenIdOfOwner(address _owner) public view returns (uint256) {
        require(balanceOf(_owner) > 0, "NOT_MINTED");
        return _tokenIdOfOwner(_owner);
    }

    function _tokenIdOfOwner(address _owner) internal pure returns (uint256) {
        return uint256(uint160(_owner));
    }

    /**
     * @notice  Returns decimal tokenid for a given address
     * @return  uint256 decimal tokenId
     */
    function baseURI() public view returns (string memory) {
        return
            string(
                abi.encodePacked(
                    attestationStation.attestations(
                        owner(),
                        address(this),
                        bytes32("optimist.base-uri")
                    )
                )
            );
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
        require(_from == address(0) || _to == address(0), "SBT_TRANSFER");
    }

    /**
     * @notice  Soulbound
     * @dev     Override internal function to prevent transfers of the Optimist token.
     */
    function approve(address, uint256) public pure override {
        revert("SBT_APPROVE");
    }

    /**
     * @notice  Soulbound
     * @dev     Override internal function to prevent transfers of the Optimist token.
     */
    function _setApprovalForAll(
        address,
        address,
        bool
    ) internal virtual override {
        revert("SBT_SET_APPROVAL_FOR_ALL");
    }
}

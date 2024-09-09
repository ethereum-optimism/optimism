// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { ERC721BurnableUpgradeable } from
    "@openzeppelin/contracts-upgradeable/token/ERC721/extensions/ERC721BurnableUpgradeable.sol";
import { AttestationStation } from "src/periphery/op-nft/AttestationStation.sol";
import { OptimistAllowlist } from "src/periphery/op-nft/OptimistAllowlist.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";

/// @author Optimism Collective
/// @author Gitcoin
/// @title  Optimist
/// @notice A Soul Bound Token for real humans only(tm).
contract Optimist is ERC721BurnableUpgradeable, ISemver {
    /// @notice Attestation key used by the attestor to attest the baseURI.
    bytes32 public constant BASE_URI_ATTESTATION_KEY = bytes32("optimist.base-uri");

    /// @notice Attestor who attests to baseURI.
    address public immutable BASE_URI_ATTESTOR;

    /// @notice Address of the AttestationStation contract.
    AttestationStation public immutable ATTESTATION_STATION;

    /// @notice Address of the OptimistAllowlist contract.
    OptimistAllowlist public immutable OPTIMIST_ALLOWLIST;

    /// @notice Semantic version.
    /// @custom:semver 2.1.1-beta.1
    string public constant version = "2.1.1-beta.1";

    /// @param _name               Token name.
    /// @param _symbol             Token symbol.
    /// @param _baseURIAttestor    Address of the baseURI attestor.
    /// @param _attestationStation Address of the AttestationStation contract.
    /// @param _optimistAllowlist  Address of the OptimistAllowlist contract
    constructor(
        string memory _name,
        string memory _symbol,
        address _baseURIAttestor,
        AttestationStation _attestationStation,
        OptimistAllowlist _optimistAllowlist
    ) {
        BASE_URI_ATTESTOR = _baseURIAttestor;
        ATTESTATION_STATION = _attestationStation;
        OPTIMIST_ALLOWLIST = _optimistAllowlist;
        initialize(_name, _symbol);
    }

    /// @notice Initializes the Optimist contract.
    /// @param _name   Token name.
    /// @param _symbol Token symbol.
    function initialize(string memory _name, string memory _symbol) public initializer {
        __ERC721_init(_name, _symbol);
        __ERC721Burnable_init();
    }

    /// @notice Allows an address to mint an Optimist NFT. Token ID is the uint256 representation
    ///         of the recipient's address. Recipients must be permitted to mint, eventually anyone
    ///         will be able to mint. One token per address.
    /// @param _recipient Address of the token recipient.
    function mint(address _recipient) public {
        require(isOnAllowList(_recipient), "Optimist: address is not on allowList");
        _safeMint(_recipient, tokenIdOfAddress(_recipient));
    }

    /// @notice Returns the baseURI for all tokens.
    /// @return uri_ BaseURI for all tokens.
    function baseURI() public view returns (string memory uri_) {
        uri_ = string(
            abi.encodePacked(
                ATTESTATION_STATION.attestations(BASE_URI_ATTESTOR, address(this), bytes32("optimist.base-uri"))
            )
        );
    }

    /// @notice Returns the token URI for a given token by ID
    /// @param _tokenId Token ID to query.
    /// @return uri_ Token URI for the given token by ID.
    function tokenURI(uint256 _tokenId) public view virtual override returns (string memory uri_) {
        uri_ = string(
            abi.encodePacked(
                baseURI(),
                "/",
                // Properly format the token ID as a 20 byte hex string (address).
                Strings.toHexString(_tokenId, 20),
                ".json"
            )
        );
    }

    /// @notice Checks OptimistAllowlist to determine whether a given address is allowed to mint
    ///         the Optimist NFT. Since the Optimist NFT will also be used as part of the
    ///         Citizens House, mints are currently restricted. Eventually anyone will be able
    ///         to mint.
    /// @return allowed_ Whether or not the address is allowed to mint yet.
    function isOnAllowList(address _recipient) public view returns (bool allowed_) {
        allowed_ = OPTIMIST_ALLOWLIST.isAllowedToMint(_recipient);
    }

    /// @notice Returns the token ID for the token owned by a given address. This is the uint256
    ///         representation of the given address.
    /// @return Token ID for the token owned by the given address.
    function tokenIdOfAddress(address _owner) public pure returns (uint256) {
        return uint256(uint160(_owner));
    }

    /// @notice Disabled for the Optimist NFT (Soul Bound Token).
    function approve(address, uint256) public pure override {
        revert("Optimist: soul bound token");
    }

    /// @notice Disabled for the Optimist NFT (Soul Bound Token).
    function setApprovalForAll(address, bool) public virtual override {
        revert("Optimist: soul bound token");
    }

    /// @notice Prevents transfers of the Optimist NFT (Soul Bound Token).
    /// @param _from Address of the token sender.
    /// @param _to   Address of the token recipient.
    function _beforeTokenTransfer(address _from, address _to, uint256) internal virtual override {
        require(_from == address(0) || _to == address(0), "Optimist: soul bound token");
    }
}

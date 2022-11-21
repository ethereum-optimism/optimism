// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "@openzeppelin/contracts-upgradeable/token/ERC721/ERC721Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC721/extensions/ERC721BurnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "./AttestationStation.sol";

contract Optimist is
    Initializable,
    ERC721Upgradeable,
    ERC721BurnableUpgradeable,
    OwnableUpgradeable
{
    AttestationStation public attestationStation;
    address public admin;

    /**
     * @notice  Initialize the Optimist contract.
     * @dev     Initializes the Optimist contract with the given parameters.
     * @param   _name  The token name.
     * @param   _symbol  The token symbol.
     * @param   _admin  The administrator address.
     * @param   _attestationStation  The address of the attestation station contract.
     */
    function initialize(
        string calldata _name,
        string calldata _symbol,
        address _admin,
        address _attestationStation
    ) public initializer {
        __ERC721_init(_name, _symbol);
        __ERC721Burnable_init();
        __Ownable_init();
        attestationStation = AttestationStation(_attestationStation);
        admin = _admin;
    }

    /**
     * @notice  Mint the Optimist token.
     * @dev     Mints the Optimist token to the give recipient address.
     *          Limits the number of tokens that can be minted to one per address.
     *          The tokenId is the uint256 of the recipient address.
     * @param   _recipient  The address of the token recipient.
     */
    function mint(address _recipient) public {
        require(balanceOf(_recipient) == 0, "Optimist::mint: ALREADY_MINTED");
        uint256 tokenId = uint256(uint160(_recipient));
        _safeMint(_recipient, tokenId);
    }

    /**
     * @notice  (Internal) Optimist Token Base URI.
     * @dev     Returns the base URI for the Optimist token from the attestation.
     * @return  string  The token URI.
     */
    function _baseURI() internal view override returns (string memory) {
        return
            string(
                attestationStation.attestations(
                    admin,
                    address(this),
                    keccak256("opnft.optimistNftBaseURI")
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
    ) internal pure override {
        require(
            _from == address(0) || _to == address(0),
            "Optimist::_beforeTokenTransfer: SOUL_BOUND"
        );
    }
}

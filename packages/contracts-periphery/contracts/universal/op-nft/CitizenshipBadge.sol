// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "@rari-capital/solmate/src/tokens/ERC721.sol";
import "@openzeppelin/contracts/utils/Strings.sol";
import "./CitizenshipChecker.sol";
import "./SocialContract.sol";

contract CitizenshipBadge is ERC721 {
    address public admin;

    SocialContract public socialContract;
    CitizenshipChecker public citizenshipChecker;

    constructor(
        address _admin,
        address _socialContract,
        address _citizenChecker
    ) ERC721("Citizenship", "CB") {
        admin = _admin;
        socialContract = SocialContract(_socialContract);
        citizenshipChecker = CitizenshipChecker(_citizenChecker);
    }

    function mint(address _to, bytes memory _proof) public returns (uint256) {
        if (!citizenshipChecker.isCitizen(_to, _proof)) {
            revert("CitizenshipBadge::mint: NOT_CITIZEN");
        }
        uint256 newItemId = uint256(uint160(_to));
        _safeMint(_to, newItemId);
        return newItemId;
    }

    function transferFrom(
        address,
        address,
        uint256
    ) public virtual override {
        revert("CitizenshipBadge::transferFrom: SOUL_BOUND");
    }

    function tokenURI(uint256 tokenId) public view virtual override returns (string memory) {
        if (ownerOf(tokenId) == address(0)) {
            revert("CitizenshipBadge:::tokenURI: TOKEN_URI_DNE");
        }
        return Strings.toString(uint256(uint160(ownerOf(tokenId))));
    }

    function baseURI() public view returns (bytes memory) {
        return
            socialContract.attestations(
                admin,
                address(this),
                keccak256("opnft.citizenshipBadgeNftBaseURI")
            );
    }
}

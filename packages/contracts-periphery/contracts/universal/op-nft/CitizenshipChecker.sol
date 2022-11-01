// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SocialContract } from "./SocialContract.sol";

interface ICitizenshipChecker {
    function isCitizen(address _who, bytes memory _proof) external view returns (bool);
}

contract CitizenshipChecker is ICitizenshipChecker {
    address public immutable root;

    SocialContract public immutable sc;

    constructor(address _root, address sca) {
        root = _root;
        sc = SocialContract(sca);
    }

    function isCitizen(address _who, bytes memory _proof) public view returns (bool) {
        (address opco, uint256 index) = abi.decode(_proof, (address, uint256));
        bytes memory numOpcoCitizenships = sc.attestations(root, opco, keccak256("op.opco"));
        require(
            index < abi.decode(numOpcoCitizenships, (uint256)),
            "CitizenshipChecker::isCitizen: INVALID_INDEX"
        );
        bytes memory citizenship = sc.attestations(
            opco,
            _who,
            keccak256(abi.encodePacked("op.opco.citizen", index))
        );
        require(abi.decode(citizenship, (bool)), "CitizenshipChecker::isCitizen: NOT_CITIZEN");
        return true;
    }
}

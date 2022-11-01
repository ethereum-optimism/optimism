//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import "forge-std/Test.sol";
import { Optimist } from "../../universal/op-nft/Optimist.sol";
import { SocialContract } from "../../universal/op-nft/SocialContract.sol";

contract OptimistTest is Test {
    using stdStorage for StdStorage;

    Optimist private optimistNFT;
    SocialContract private socialContract;

    address alice = address(1);
    address admin = address(2);

    function _setUp() public {
        socialContract = new SocialContract();
    }

    function test_optimistNFTDeployed() public {
        optimistNFT = new Optimist("OPTIMIST", "OPT", admin, address(socialContract));
        assertEq(optimistNFT.name(), "OPTIMIST");
    }

    function test_noBalance() public {
        optimistNFT = new Optimist("OPTIMIST", "OPT", admin, address(socialContract));
        assertEq(optimistNFT.balanceOf(alice), 0);
    }

    function test_mint() public {
        optimistNFT = new Optimist("OPTIMIST", "OPT", admin, address(socialContract));
        optimistNFT.mint(alice);
        assertEq(optimistNFT.balanceOf(alice), 1);
    }

    function test_balanceIncremented() public {
        optimistNFT = new Optimist("OPTIMIST", "OPT", admin, address(socialContract));
        optimistNFT.mint(alice);
        uint256 slotBalance = stdstore
            .target(address(optimistNFT))
            .sig(optimistNFT.balanceOf.selector)
            .with_key(alice)
            .find();

        uint256 balanceAfterMint = uint256(vm.load(address(optimistNFT), bytes32(slotBalance)));

        assertEq(balanceAfterMint, 1);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { OptimismMintableERC20 } from "src/universal/OptimismMintableERC20.sol";
import { LegacyMintableERC20 } from "src/legacy/LegacyMintableERC20.sol";

/// @title Bridge_Initializer
/// @dev This contract extends the CommonTest contract with token deployments
///      meant to be used with the bridge contracts.
contract Bridge_Initializer is CommonTest {
    ERC20 L1Token;
    ERC20 BadL1Token;
    OptimismMintableERC20 L2Token;
    LegacyMintableERC20 LegacyL2Token;
    ERC20 NativeL2Token;
    ERC20 BadL2Token;
    OptimismMintableERC20 RemoteL1Token;

    function setUp() public virtual override {
        super.setUp();

        L1Token = new ERC20("Native L1 Token", "L1T");

        LegacyL2Token = new LegacyMintableERC20({
            _l2Bridge: address(l2StandardBridge),
            _l1Token: address(L1Token),
            _name: string.concat("LegacyL2-", L1Token.name()),
            _symbol: string.concat("LegacyL2-", L1Token.symbol())
        });
        vm.label(address(LegacyL2Token), "LegacyMintableERC20");

        // Deploy the L2 ERC20 now
        L2Token = OptimismMintableERC20(
            l2OptimismMintableERC20Factory.createStandardL2Token(
                address(L1Token),
                string(abi.encodePacked("L2-", L1Token.name())),
                string(abi.encodePacked("L2-", L1Token.symbol()))
            )
        );

        BadL2Token = OptimismMintableERC20(
            l2OptimismMintableERC20Factory.createStandardL2Token(
                address(1),
                string(abi.encodePacked("L2-", L1Token.name())),
                string(abi.encodePacked("L2-", L1Token.symbol()))
            )
        );

        NativeL2Token = new ERC20("Native L2 Token", "L2T");

        RemoteL1Token = OptimismMintableERC20(
            l1OptimismMintableERC20Factory.createStandardL2Token(
                address(NativeL2Token),
                string(abi.encodePacked("L1-", NativeL2Token.name())),
                string(abi.encodePacked("L1-", NativeL2Token.symbol()))
            )
        );

        BadL1Token = OptimismMintableERC20(
            l1OptimismMintableERC20Factory.createStandardL2Token(
                address(1),
                string(abi.encodePacked("L1-", NativeL2Token.name())),
                string(abi.encodePacked("L1-", NativeL2Token.symbol()))
            )
        );
    }
}

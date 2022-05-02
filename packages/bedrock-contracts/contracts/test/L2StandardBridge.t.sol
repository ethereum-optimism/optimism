//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import {
    Lib_PredeployAddresses
} from "@eth-optimism/contracts/libraries/constants/Lib_PredeployAddresses.sol";

import { IWithdrawer } from "../L2/IWithdrawer.sol";
import { Withdrawer } from "../L2/Withdrawer.sol";
import { L2StandardBridge } from "../L2/messaging/L2StandardBridge.sol";
import { L1StandardBridge } from "../L1/messaging/L1StandardBridge.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { Lib_BedrockPredeployAddresses } from "../libraries/Lib_BedrockPredeployAddresses.sol";
import { L2StandardTokenFactory } from "../L2/messaging/L2StandardTokenFactory.sol";
import { IL2StandardTokenFactory } from "../L2/messaging/IL2StandardTokenFactory.sol";
import { L2StandardERC20 } from "../L2/tokens/L2StandardERC20.sol";
import { IL2StandardERC20 } from "../L2/tokens/IL2StandardERC20.sol";

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { CommonTest } from "./CommonTest.t.sol";
import { L2OutputOracle_Initializer } from "./L2OutputOracle.t.sol";
import { LibRLP } from "./Lib_RLP.t.sol";

import { console } from "forge-std/console.sol";

contract L2StandardBridge_Test is CommonTest, L2OutputOracle_Initializer {
    OptimismPortal op;

    IWithdrawer W;
    L1StandardBridge L1Bridge;
    L2StandardBridge L2Bridge;
    IL2StandardTokenFactory L2TokenFactory;
    IL2StandardERC20 L2Token;

    function setUp() external {
        L1Bridge = new L1StandardBridge();
        L2Bridge = new L2StandardBridge(address(L1Bridge));
        op = new OptimismPortal(oracle, 100);

        L1Bridge.initialize(op, address(L2Bridge));

        Withdrawer w = new Withdrawer();
        vm.etch(Lib_BedrockPredeployAddresses.WITHDRAWER, address(w).code);
        W = IWithdrawer(Lib_BedrockPredeployAddresses.WITHDRAWER);

        L2StandardTokenFactory factory = new L2StandardTokenFactory();
        vm.etch(Lib_PredeployAddresses.L2_STANDARD_TOKEN_FACTORY, address(factory).code);
        L2TokenFactory = IL2StandardTokenFactory(Lib_PredeployAddresses.L2_STANDARD_TOKEN_FACTORY);

        ERC20 token = new ERC20("Test Token", "TT");

        // Deploy the L2 ERC20 now
        L2TokenFactory.createStandardL2Token(
            address(token),
            string(abi.encodePacked("L2-", token.name())),
            string(abi.encodePacked("L2-", token.symbol()))
        );

        L2Token = IL2StandardERC20(
            LibRLP.computeAddress(address(L2TokenFactory), 0)
        );
    }

    function test_L2BridgeCorrectL1Bridge() external {
        address l1Bridge = L2Bridge.l1TokenBridge();
        assertEq(address(L1Bridge), l1Bridge);
    }

    // withdraw
    // - token is burned
    // - emits WithdrawalInitiated
    // - calls Withdrawer.initiateWithdrawal
    // withdrawTo
    // - token is burned
    // - emits WithdrawalInitiated w/ correct recipient
    // - calls Withdrawer.initiateWithdrawal
    // finalizeDeposit
    // - only callable by l1TokenBridge
    // - supported token pair emits DepositFinalized
    // - invalid deposit emits DepositFailed
    // - invalid deposit calls Withdrawer.initiateWithdrawal
}


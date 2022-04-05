//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { DepositFeed } from "./DepositFeed.sol";
import { WithdrawalVerifier } from "./WithdrawalVerifier.sol";
import { L2OutputOracle } from "./L2OutputOracle.sol";


contract OptimismPortal is DepositFeed, WithdrawalVerifier {
    constructor(L2OutputOracle _l2Oracle, uint256 _finalizationWindow)
        WithdrawalVerifier(_l2Oracle, _finalizationWindow)
    {}


}

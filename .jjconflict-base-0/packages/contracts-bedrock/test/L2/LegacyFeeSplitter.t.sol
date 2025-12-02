// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { FeeSplitter_TestInit } from "test/L2/FeeSplitter.t.sol";
import { LegacyFeeSplitter } from "test/mocks/LegacyFeeSplitter.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title LegacyFeeSplitter_DisburseFees_Test
/// @notice Test contract for the LegacyFeeSplitter contract's functionality
contract LegacyFeeSplitter_DisburseFees_Test is FeeSplitter_TestInit {
    LegacyFeeSplitter public legacyFeeSplitter;

    function setUp() public override {
        super.setUp();

        legacyFeeSplitter = new LegacyFeeSplitter();

        // Setup the legacy splitter as the recipient in the vaults
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        vm.startPrank(owner);
        IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).setRecipient(address(legacyFeeSplitter));
        IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).setRecipient(address(legacyFeeSplitter));
        IFeeVault(payable(Predeploys.L1_FEE_VAULT)).setRecipient(address(legacyFeeSplitter));
        IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).setRecipient(address(legacyFeeSplitter));
        vm.stopPrank();
    }

    function test_legacyFeeSplitterDisburseFees_succeeds(
        uint256 _sequencerBalance,
        uint256 _baseBalance,
        uint256 _l1Balance,
        uint256 _operatorBalance
    )
        public
    {
        _sequencerBalance = bound(
            _sequencerBalance,
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).minWithdrawalAmount(),
            type(uint128).max
        );

        _baseBalance =
            bound(_baseBalance, IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).minWithdrawalAmount(), type(uint128).max);

        _l1Balance =
            bound(_l1Balance, IFeeVault(payable(Predeploys.L1_FEE_VAULT)).minWithdrawalAmount(), type(uint128).max);

        _operatorBalance = bound(
            _operatorBalance, IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).minWithdrawalAmount(), type(uint128).max
        );

        // Setup mock fee vaults
        _mockFeeVaultForSuccessfulWithdrawalWithSplitter(
            address(legacyFeeSplitter), Predeploys.SEQUENCER_FEE_WALLET, uint256(_sequencerBalance)
        );
        _mockFeeVaultForSuccessfulWithdrawalWithSplitter(
            address(legacyFeeSplitter), Predeploys.BASE_FEE_VAULT, uint256(_baseBalance)
        );
        _mockFeeVaultForSuccessfulWithdrawalWithSplitter(
            address(legacyFeeSplitter), Predeploys.L1_FEE_VAULT, uint256(_l1Balance)
        );
        _mockFeeVaultForSuccessfulWithdrawalWithSplitter(
            address(legacyFeeSplitter), Predeploys.OPERATOR_FEE_VAULT, uint256(_operatorBalance)
        );

        assertEq(address(legacyFeeSplitter).balance, 0);
        legacyFeeSplitter.disburseFees();
        assertEq(address(legacyFeeSplitter).balance, _sequencerBalance + _baseBalance + _l1Balance + _operatorBalance);
    }
}

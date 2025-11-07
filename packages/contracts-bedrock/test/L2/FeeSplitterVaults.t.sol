// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Testing
import { Test } from "forge-std/Test.sol";
import { FeeSplitterForTest } from "test/mocks/FeeSplitterForTest.sol";

// Interfaces
import { IFeeSplitter } from "interfaces/L2/IFeeSplitter.sol";

/// @title FeeSplitterVaults_Test
/// @notice Test contract for the FeeSplitter contract with vaults.
/// @dev This test is done in a different file given we need the 0.8.25 compiler version to import the FeeSplitter
/// implementation and modify it.
contract FeeSplitterVaults_Receive_Test is Test {
    FeeSplitterForTest feeSplitter;
    address[4] internal _feeVaults;

    function setUp() public {
        FeeSplitterForTest feeSplitterImpl = new FeeSplitterForTest();
        feeSplitter = FeeSplitterForTest(payable(Predeploys.FEE_SPLITTER));

        vm.etch(Predeploys.FEE_SPLITTER, address(feeSplitterImpl).code);
        vm.etch(Predeploys.SEQUENCER_FEE_WALLET, vm.getDeployedCode("SequencerFeeVault.sol"));
        vm.etch(Predeploys.BASE_FEE_VAULT, vm.getDeployedCode("BaseFeeVault.sol"));
        vm.etch(Predeploys.OPERATOR_FEE_VAULT, vm.getDeployedCode("OperatorFeeVault.sol"));
        vm.etch(Predeploys.L1_FEE_VAULT, vm.getDeployedCode("L1FeeVault.sol"));

        _feeVaults[0] = Predeploys.SEQUENCER_FEE_WALLET;
        _feeVaults[1] = Predeploys.BASE_FEE_VAULT;
        _feeVaults[2] = Predeploys.OPERATOR_FEE_VAULT;
        _feeVaults[3] = Predeploys.L1_FEE_VAULT;
    }

    /// @notice Test that receive function reverts when sender is an approved vault but not currently disbursing
    /// @param _amount The amount of ETH to send.
    /// @dev This test simulates the disbursement context on each vault and then goes
    /// through each vault and sends ETH to the fee splitter, expecting the receive function to revert
    function testFuzz_feeSplitterReceive_whenNotCurrentVault_reverts(uint128 _amount) public {
        for (uint256 i = 0; i < _feeVaults.length; i++) {
            address _disbursingVault = _feeVaults[i];
            feeSplitter.setTransientDisbursingAddress(_disbursingVault);
            for (uint256 j = 0; j < _feeVaults.length; j++) {
                address _selectedVault = _feeVaults[j];
                vm.deal(_selectedVault, _amount);

                if (_selectedVault != _disbursingVault) {
                    vm.expectRevert(IFeeSplitter.FeeSplitter_SenderNotCurrentVault.selector);
                }

                vm.prank(_selectedVault);
                payable(address(feeSplitter)).call{ value: _amount }("");
            }
        }
    }
}

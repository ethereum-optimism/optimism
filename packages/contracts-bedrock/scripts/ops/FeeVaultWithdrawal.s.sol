// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console } from "forge-std/console.sol";
import { Script } from "forge-std/Script.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IFeeVault } from "src/L2/interfaces/IFeeVault.sol";

/// @title FeeVaultWithdrawal
/// @notice A script to make it very simple to withdraw from the fee vaults.
///         The usage is as follows:
///         $ forge script scripts/FeeVaultWithdrawal.s.sol \
///             --rpc-url $ETH_RPC_URL --broadcast \
///             --private-key $PRIVATE_KEY
contract FeeVaultWithdrawal is Script {
    IMulticall3 private constant multicall = IMulticall3(MULTICALL3_ADDRESS);
    IMulticall3.Call3[] internal calls;

    /// @notice The entrypoint function. Determines which FeeVaults can be withdrawn from and then
    ///        will send the transaction via Multicall3 to withdraw all FeeVaults.
    function run() external {
        require(address(multicall).code.length > 0);

        address[] memory vaults = new address[](3);
        vaults[0] = Predeploys.SEQUENCER_FEE_WALLET;
        vaults[1] = Predeploys.BASE_FEE_VAULT;
        vaults[2] = Predeploys.L1_FEE_VAULT;

        for (uint256 i; i < vaults.length; i++) {
            address vault = vaults[i];
            bool shouldCall = canWithdrawal(vault);
            if (shouldCall) {
                calls.push(
                    IMulticall3.Call3({
                        target: vault,
                        allowFailure: false,
                        callData: abi.encodeWithSelector(IFeeVault.withdraw.selector)
                    })
                );

                address recipient = IFeeVault(payable(vault)).RECIPIENT();
                uint256 balance = vault.balance;
                log(balance, recipient, vault);
            } else {
                string memory logline =
                    string.concat(vm.toString(vault), " does not have a large enough balance to withdraw.");
                console.log(logline);
            }
        }

        if (calls.length > 0) {
            vm.broadcast();
            multicall.aggregate3(calls);
            console.log("Success.");
        }
    }

    /// @notice Checks whether or not a FeeVault can be withdrawn. The balance of the account must
    ///         be larger than the `MIN_WITHDRAWAL_AMOUNT`.
    function canWithdrawal(address _vault) internal view returns (bool) {
        uint256 minWithdrawalAmount = IFeeVault(payable(_vault)).MIN_WITHDRAWAL_AMOUNT();
        uint256 balance = _vault.balance;
        return balance >= minWithdrawalAmount;
    }

    /// @notice Logs the information relevant to the user.
    function log(uint256 _balance, address _recipient, address _vault) internal pure {
        string memory logline = string.concat(
            "Withdrawing ", vm.toString(_balance), " to ", vm.toString(_recipient), " from ", vm.toString(_vault)
        );
        console.log(logline);
    }
}

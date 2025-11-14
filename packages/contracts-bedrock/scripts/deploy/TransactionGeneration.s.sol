// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Script } from "forge-std/Script.sol";
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { L2ContractsManager } from "src/L2/L2ContractsManager.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Config, Fork } from "scripts/libraries/Config.sol";
import { console2 as console } from "forge-std/console2.sol";
import { PredeployHelper } from "scripts/deploy/PredeployHelper.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { ICreate2Deployer } from "interfaces/preinstalls/ICreate2Deployer.sol";
import { L2ImplementationsDeployer } from "src/L2/L2ImplementationsDeployer.sol";

/// @title TransactionGenerationScript
/// @notice Script that generates Network Upgrade Transactions (NUTs) for deploying L2 contracts during a hard fork.
///         This script creates a sequence of transactions that deploy Predeploy contracts using CREATE2 and execute
///         and the L2ContractsManager. The last transaction is the execution of the L2ContractsManager.
contract TransactionGeneration is Script {
    /// @notice Address of the Create2Deployer predeploy.
    address payable immutable CREATE2_DEPLOYER = payable(Preinstalls.Create2Deployer);

    /// @notice Array of Network Upgrade Transactions.
    NetworkUpgradeTxns.NetworkUpgradeTxn[] private txns;

    /// @notice Helper for managing predeploy configurations.
    PredeployHelper internal helper;

    /// @notice Address of the L2ImplementationsDeployer contract.
    address private l2ImplDeployerAddress;

    /// @notice Input struct for the script.
    /// @param l2ChainID The ID of the L2 chain.
    /// @param l1ChainID The ID of the L1 chain.
    /// @param l1CrossDomainMessengerProxy The address of the L1 Cross Domain Messenger proxy.
    /// @param l1StandardBridgeProxy The address of the L1 Standard Bridge proxy.
    /// @param l1ERC721BridgeProxy The address of the L1 ERC721 Bridge proxy.
    /// @param opChainProxyAdminOwner The address of the OP Chain Proxy Admin owner.
    /// @param sequencerFeeVaultRecipient The address of the Sequencer Fee Vault recipient.
    /// @param sequencerFeeVaultMinimumWithdrawalAmount The minimum withdrawal amount for the Sequencer Fee Vault.
    /// @param sequencerFeeVaultWithdrawalNetwork The withdrawal network for the Sequencer Fee Vault.
    /// @param baseFeeVaultRecipient The address of the Base Fee Vault recipient.
    /// @param baseFeeVaultMinimumWithdrawalAmount The minimum withdrawal amount for the Base Fee Vault.
    /// @param baseFeeVaultWithdrawalNetwork The withdrawal network for the Base Fee Vault.
    /// @param l1FeeVaultRecipient The address of the L1 Fee Vault recipient.
    /// @param l1FeeVaultMinimumWithdrawalAmount The minimum withdrawal amount for the L1 Fee Vault.
    /// @param l1FeeVaultWithdrawalNetwork The withdrawal network for the L1 Fee Vault.
    /// @param l2ImplDeployerAddress The address of the already-deployed L2ImplementationsDeployer.
    /// @param l2cmName The name of the L2 Contracts Manager.
    struct Input {
        uint256 l2ChainID;
        uint256 l1ChainID;
        address payable l1CrossDomainMessengerProxy;
        address payable l1StandardBridgeProxy;
        address payable l1ERC721BridgeProxy;
        address opChainProxyAdminOwner;
        address sequencerFeeVaultRecipient;
        uint256 sequencerFeeVaultMinimumWithdrawalAmount;
        uint256 sequencerFeeVaultWithdrawalNetwork;
        address baseFeeVaultRecipient;
        uint256 baseFeeVaultMinimumWithdrawalAmount;
        uint256 baseFeeVaultWithdrawalNetwork;
        address l1FeeVaultRecipient;
        uint256 l1FeeVaultMinimumWithdrawalAmount;
        uint256 l1FeeVaultWithdrawalNetwork;
        address l2ImplDeployerAddress;
        string l2cmName;
        string hardForkName;
    }

    /// @notice Output struct for the script
    /// @param txns Array of Network Upgrade Transactions generated
    /// @param l2cmAddress Address where the L2ContractsManager is deployed
    /// @param predeploys Array of predeploys that were changed
    struct Output {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] txns;
        address l2cmAddress;
        PredeployHelper.Predeploy[] predeploys;
    }

    /// @notice Generates Network Upgrade Transactions for deploying L2 contracts during a hard fork
    /// @dev Creates a sequence of transactions that:
    ///      1. Deploy new predeploy implementations via L2ImplementationsDeployer
    ///      2. Deploy the L2ContractsManager via CREATE2
    ///      3. Execute the L2ContractsManager to upgrade all predeploy proxies
    ///      The final artifact is written to deployments/nut-xfork-upgrade-transactions.json
    /// @param _input The input struct containing chain configuration and deployment parameters
    /// @return Output struct containing the generated transactions, L2CM address, and changed predeploys
    function run(Input memory _input) external returns (Output memory) {
        helper = new PredeployHelper();

        // Set the L2ImplementationsDeployer address from input
        l2ImplDeployerAddress = _input.l2ImplDeployerAddress;

        // Get all changed predeploy implementations
        PredeployHelper.Predeploy[] memory predeploys = helper.getPredeploys(_input);

        // Generate deployment transactions for each changed predeploy implementations
        generateDeploymentTransactions(_input.hardForkName, predeploys);

        address[] memory predeploysAddresses = new address[](predeploys.length);
        for (uint256 i = 0; i < predeploysAddresses.length; i++) {
            predeploysAddresses[i] = predeploys[i].implementation;
        }
        // Generate the L2ContractsManager deployment transaction
        address l2cmAddress = generateL2ContractsManagerDeploymentTransaction(_input, predeploysAddresses);

        // Generate L2ContractsManager execute transaction
        generateL2ContractsManagerExecuteTransaction(_input.hardForkName, _input.l2cmName, l2cmAddress, predeploys);

        // Write all transactions to JSON artifact file
        NetworkUpgradeTxns.writeArtifact(
            txns, string.concat("deployments/nut-", _input.hardForkName, "-upgrade-transactions.json")
        );

        return Output({ txns: txns, l2cmAddress: l2cmAddress, predeploys: predeploys });
    }

    /// @notice Generates deployment transactions for all changed predeploys using L2ImplementationsDeployer
    /// @dev Each predeploy is deployed via the L2ImplementationsDeployer with a salt derived from its name
    /// @param predeploys Array of predeploys that need to be deployed
    function generateDeploymentTransactions(
        string memory _hardForkName,
        PredeployHelper.Predeploy[] memory predeploys
    )
        internal
    {
        for (uint256 i = 0; i < predeploys.length; i++) {
            txns.push(
                NetworkUpgradeTxns.newTx({
                    intent: string.concat(_hardForkName, predeploys[i].name, " Deployment"),
                    from: address(0),
                    to: l2ImplDeployerAddress,
                    mint: 0,
                    value: 0,
                    gas: 1_000_000_000,
                    isSystemTransaction: false,
                    data: abi.encodeCall(
                        L2ImplementationsDeployer.deploy,
                        (0, keccak256(abi.encode(predeploys[i].name)), predeploys[i].initCode)
                    )
                })
            );
        }
    }

    /// @notice Generates a deployment transaction for the L2ContractsManager using L2ImplementationsDeployer
    /// @dev The L2ContractsManager is deployed via the L2ImplementationsDeployer with a salt derived from its name
    function generateL2ContractsManagerDeploymentTransaction(
        TransactionGeneration.Input memory _input,
        address[] memory predeploysAddresses
    )
        internal
        returns (address l2cmAddress)
    {
        bytes memory constructorArgs = _encodeL2CMConstructorArgs(predeploysAddresses);
        bytes memory initCode = abi.encodePacked(vm.getCode(_input.l2cmName), constructorArgs);
        bytes32 salt = keccak256(abi.encode(_input.hardForkName, _input.l2cmName));

        l2cmAddress = ICreate2Deployer(CREATE2_DEPLOYER).computeAddress(salt, keccak256(initCode));

        // Generate the L2ContractsManager deployment transaction
        txns.push(
            NetworkUpgradeTxns.newTx({
                intent: string.concat(_input.hardForkName, ": ", _input.l2cmName, " Deployment"),
                from: address(0),
                to: l2ImplDeployerAddress,
                mint: 0,
                value: 0,
                gas: 1_000_000,
                isSystemTransaction: false,
                data: abi.encodeCall(L2ImplementationsDeployer.deploy, (0, salt, initCode))
            })
        );
    }

    /// @notice Helper function to encode constructor arguments for L2ContractsManager
    /// @dev Uses assembly to avoid stack too deep errors
    function _encodeL2CMConstructorArgs(address[] memory addrs) internal pure returns (bytes memory result) {
        result = new bytes(32 * 17);
        assembly {
            let ptr := add(result, 32)
            for { let i := 0 } lt(i, 17) { i := add(i, 1) } {
                let addr := mload(add(add(addrs, 32), mul(i, 32)))
                mstore(add(ptr, mul(i, 32)), addr)
            }
        }
    }

    /// @notice Generates a transaction that executes the L2ContractsManager via ProxyAdmin to upgrade predeploys
    /// @dev The transaction calls ProxyAdmin.performDelegateCall to delegatecall into the L2ContractsManager,
    ///      which upgrades all predeploy proxies to their new implementations
    /// @param _hardForkName The name of the hard fork
    /// @param _l2cmName The name of the L2ContractsManager contract
    /// @param _l2cmAddress The address where the L2ContractsManager is deployed
    /// @param predeploys Array of predeploys that were deployed and need to be upgraded
    function generateL2ContractsManagerExecuteTransaction(
        string memory _hardForkName,
        string memory _l2cmName,
        address _l2cmAddress,
        PredeployHelper.Predeploy[] memory predeploys
    )
        internal
    {
        // Build the ProxyUpgrade array for the L2ContractsManager
        L2ContractsManager.ProxyUpgrade[] memory proxyUpgrades =
            new L2ContractsManager.ProxyUpgrade[](predeploys.length);
        for (uint256 i = 0; i < predeploys.length; i++) {
            proxyUpgrades[i] = L2ContractsManager.ProxyUpgrade({
                proxy: predeploys[i].proxy,
                implementation: predeploys[i].implementation
            });
        }

        // Create transaction that calls execute() on the deployed L2ContractsManager
        txns.push(
            NetworkUpgradeTxns.newTx({
                intent: string.concat(_hardForkName, ": ", _l2cmName, " Execute"),
                from: Constants.DEPOSITOR_ACCOUNT,
                to: Predeploys.PROXY_ADMIN,
                mint: 0,
                value: 0,
                gas: type(uint64).max,
                isSystemTransaction: false,
                data: abi.encodeCall(ProxyAdmin.performDelegateCall, (_l2cmAddress))
            })
        );
    }
}

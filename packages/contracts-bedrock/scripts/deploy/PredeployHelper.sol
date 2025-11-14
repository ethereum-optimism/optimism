// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Script } from "forge-std/Script.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { console } from "forge-std/console.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { ICreate2Deployer } from "interfaces/preinstalls/ICreate2Deployer.sol";
import { TransactionGeneration } from "scripts/deploy/TransactionGeneration.s.sol";

/// @title PredeployHelper
/// @notice Helper script for managing predeploy configurations during network upgrades.
///         This contract collects all predeploys that need to be deployed, computes their
///         CREATE2 addresses, and handles special cases requiring constructor arguments.
contract PredeployHelper is Script {
    /// @notice Address of the Create2Deployer predeploy.
    address payable immutable CREATE2_DEPLOYER = payable(Preinstalls.Create2Deployer);

    /// @notice Represents a predeploy contract to be deployed during a network upgrade.
    /// @param proxy The address of the proxy contract that will be upgraded.
    /// @param name The name of the predeploy contract.
    /// @param initCode The initialization code (bytecode + constructor args) for deployment.
    /// @param implementation The computed CREATE2 address where the implementation will be deployed.
    struct Predeploy {
        address proxy;
        string name;
        bytes initCode;
        address implementation;
    }

    /// @notice Array storing all predeploys to be deployed.
    Predeploy[17] private predeploys;

    /// @notice Constructor for the PredeployHelper.
    constructor() {
        predeploys[0].proxy = Predeploys.LEGACY_MESSAGE_PASSER; // 0: LegacyMessagePasser
        predeploys[1].proxy = Predeploys.DEPLOYER_WHITELIST; // 1: DeployerWhitelist
        predeploys[2].proxy = Predeploys.L2_CROSS_DOMAIN_MESSENGER; // 2: L2CrossDomainMessenger
        predeploys[3].proxy = Predeploys.GAS_PRICE_ORACLE; // 3: GasPriceOracle
        predeploys[4].proxy = Predeploys.L2_STANDARD_BRIDGE; // 4: L2StandardBridge
        predeploys[5].proxy = Predeploys.SEQUENCER_FEE_WALLET; // 5: SequencerFeeWallet
        predeploys[6].proxy = Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY; // 6: OptimismMintableERC20Factory
        predeploys[7].proxy = Predeploys.L1_BLOCK_NUMBER; // 7: L1BlockNumber
        predeploys[8].proxy = Predeploys.L2_ERC721_BRIDGE; // 8: L2ERC721Bridge
        predeploys[9].proxy = Predeploys.L1_BLOCK_ATTRIBUTES; // 9: L1BlockAttributes
        predeploys[10].proxy = Predeploys.L2_TO_L1_MESSAGE_PASSER; // 10: L2ToL1MessagePasser
        predeploys[11].proxy = Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY; // 11: OptimismMintableERC721Factory
        predeploys[12].proxy = Predeploys.BASE_FEE_VAULT; // 12: BaseFeeVault
        predeploys[13].proxy = Predeploys.L1_FEE_VAULT; // 13: L1FeeVault
        predeploys[14].proxy = Predeploys.OPERATOR_FEE_VAULT; // 14: OperatorFeeVault
        predeploys[15].proxy = Predeploys.SCHEMA_REGISTRY; // 15: SchemaRegistry
        predeploys[16].proxy = Predeploys.EAS; // 16: EAS
    }

    /// @notice Collects all predeploys that need to be deployed for the configured fork.
    ///  @param _input The input struct containing chain configuration and deployment parameters.
    /// @return Array of Predeploy structs containing deployment information for each predeploy.
    function getPredeploys(TransactionGeneration.Input memory _input) external returns (Predeploy[] memory) {
        for (uint256 i = 0; i < predeploys.length; i++) {
            // Skip if not supported or not proxied or needs constructor args
            if (_needsConstructorArgs(predeploys[i].proxy)) {
                _addPredeploysWithArgs(_input, i, predeploys[i].proxy);
            } else {
                _addPredeploy(i, predeploys[i].proxy, bytes(""));
            }
        }

        // Copy storage array to memory for return
        Predeploy[] memory result = new Predeploy[](predeploys.length);
        for (uint256 i = 0; i < predeploys.length; i++) {
            result[i] = predeploys[i];
        }
        return result;
    }

    /// @notice Adds a predeploy to the deployment list with optional constructor arguments.
    /// @param _addr The proxy address of the predeploy contract.
    /// @param _args ABI-encoded constructor arguments (empty bytes for no-arg constructors).
    function _addPredeploy(uint256 _index, address _addr, bytes memory _args) internal {
        string memory _name = Predeploys.getName(_addr);
        bytes memory initCode = abi.encodePacked(vm.getCode(_name), _args);
        bytes32 salt = keccak256(abi.encode(_name));
        address implementation = ICreate2Deployer(CREATE2_DEPLOYER).computeAddress(salt, keccak256(initCode));

        predeploys[_index] =
            Predeploy({ proxy: _addr, name: _name, initCode: initCode, implementation: implementation });
    }

    /// @notice Checks if a predeploy requires constructor arguments or special handling.
    /// @param _proxy The address of the proxy contract to check.
    /// @return True if the predeploy requires constructor arguments, false otherwise.
    function _needsConstructorArgs(address _proxy) private pure returns (bool) {
        return _proxy == Predeploys.SEQUENCER_FEE_WALLET || _proxy == Predeploys.BASE_FEE_VAULT
            || _proxy == Predeploys.L1_FEE_VAULT || _proxy == Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY;
    }

    /// @notice Adds predeploys with constructor arguments to the deployment list.
    /// @param _input The input struct containing configuration parameters.
    function _addPredeploysWithArgs(
        TransactionGeneration.Input memory _input,
        uint256 _index,
        address _proxy
    )
        internal
    {
        if (_proxy == Predeploys.SEQUENCER_FEE_WALLET) {
            _addFeeVault(
                _index,
                Predeploys.SEQUENCER_FEE_WALLET,
                _input.sequencerFeeVaultRecipient,
                _input.sequencerFeeVaultMinimumWithdrawalAmount,
                _input.sequencerFeeVaultWithdrawalNetwork
            );
        } else if (_proxy == Predeploys.BASE_FEE_VAULT) {
            _addFeeVault(
                _index,
                Predeploys.BASE_FEE_VAULT,
                _input.baseFeeVaultRecipient,
                _input.baseFeeVaultMinimumWithdrawalAmount,
                _input.baseFeeVaultWithdrawalNetwork
            );
        } else if (_proxy == Predeploys.L1_FEE_VAULT) {
            _addFeeVault(
                _index,
                Predeploys.L1_FEE_VAULT,
                _input.l1FeeVaultRecipient,
                _input.l1FeeVaultMinimumWithdrawalAmount,
                _input.l1FeeVaultWithdrawalNetwork
            );
        } else if (_proxy == Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY) {
            _addOptimismMintableERC721Factory(_index, _input);
        }
    }

    /// @notice Adds a fee vault to the deployment list with its constructor arguments.
    /// @param _feeVault The address of the fee vault contract to add.
    /// @param _feeVaultRecipient The recipient of the fee vault.
    /// @param _feeVaultMinimumWithdrawalAmount The minimum withdrawal amount for the fee vault.
    /// @param _feeVaultWithdrawalNetwork The withdrawal network for the fee vault.
    function _addFeeVault(
        uint256 _index,
        address _feeVault,
        address _feeVaultRecipient,
        uint256 _feeVaultMinimumWithdrawalAmount,
        uint256 _feeVaultWithdrawalNetwork
    )
        internal
    {
        _addPredeploy(
            _index,
            _feeVault,
            abi.encode(_feeVaultRecipient, _feeVaultMinimumWithdrawalAmount, _feeVaultWithdrawalNetwork)
        );
    }

    /// @notice Adds the OptimismMintableERC721Factory predeploy with its constructor arguments
    /// @param _input The input struct containing configuration parameters
    function _addOptimismMintableERC721Factory(uint256 _index, TransactionGeneration.Input memory _input) internal {
        _addPredeploy(
            _index,
            Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY,
            abi.encode(_input.l1ERC721BridgeProxy, _input.l2ChainID)
        );
    }
}

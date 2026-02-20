// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Utilities
import { Script } from "forge-std/Script.sol";

// Libraries
import { Fork } from "scripts/libraries/Config.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { Constants } from "src/libraries/Constants.sol";
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";

// Interfaces
import { IProxy } from "interfaces/universal/IProxy.sol";

// Contracts
import { ConditionalDeployer } from "src/L2/ConditionalDeployer.sol";

contract GenerateNUTBundleUtils is Script {
    /// @notice Array of predeploy addresses to upgrade.
    address[] public predeploys;

    /// @notice Fork to use for the upgrade.
    Fork public fork;

    /// @notice Flag to use custom gas token for the upgrade.
    bool public useCustomGasToken;

    constructor(Fork _fork, bool _useCustomGasToken) {
        fork = _fork;
        useCustomGasToken = _useCustomGasToken;
    }

    /// @notice Returns the array of predeploy addresses to upgrade based on fork and configuration.
    /// @return predeploys_ Array of predeploy addresses to upgrade.
    function getPredeploysToUpgrade() public returns (address[] memory predeploys_) {
        // Clear previous state to avoid duplicates on reuse
        delete predeploys;

        // Always deployed predeploys (21) - StorageSetter excluded (not a predeploy)
        predeploys.push(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        predeploys.push(Predeploys.GAS_PRICE_ORACLE);
        predeploys.push(Predeploys.L2_STANDARD_BRIDGE);
        predeploys.push(Predeploys.SEQUENCER_FEE_WALLET);
        predeploys.push(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY);
        predeploys.push(Predeploys.L2_ERC721_BRIDGE);
        predeploys.push(Predeploys.L1_BLOCK_ATTRIBUTES);
        predeploys.push(Predeploys.L2_TO_L1_MESSAGE_PASSER);
        predeploys.push(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY);
        predeploys.push(Predeploys.PROXY_ADMIN);
        predeploys.push(Predeploys.BASE_FEE_VAULT);
        predeploys.push(Predeploys.L1_FEE_VAULT);
        predeploys.push(Predeploys.OPERATOR_FEE_VAULT);
        predeploys.push(Predeploys.SCHEMA_REGISTRY);
        predeploys.push(Predeploys.EAS);
        predeploys.push(Predeploys.SUPERCHAIN_ETH_BRIDGE);
        predeploys.push(Predeploys.ETH_LIQUIDITY);
        predeploys.push(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY);
        predeploys.push(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON);
        predeploys.push(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
        predeploys.push(Predeploys.FEE_SPLITTER);

        // Conditional predeploys
        if (fork >= Fork.INTEROP) {
            predeploys.push(Predeploys.CROSS_L2_INBOX);
            predeploys.push(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        }

        // TODO: review if we need to include these predeploys always
        if (useCustomGasToken) {
            predeploys.push(Predeploys.NATIVE_ASSET_LIQUIDITY);
            predeploys.push(Predeploys.LIQUIDITY_CONTROLLER);
        }

        predeploys_ = new address[](predeploys.length);
        for (uint256 i = 0; i < predeploys.length; i++) {
            predeploys_[i] = predeploys[i];
        }
    }

    /// @notice Computes the CREATE2 address for given initcode and salt.
    /// @dev Uses the DeterministicDeploymentProxy address as the deployer.
    ///      Formula: keccak256(0xff ++ deployer ++ salt ++ keccak256(initcode))[12:]
    /// @param _code The contract initcode (creation bytecode).
    /// @param _salt The CREATE2 salt.
    /// @return expected_ The computed contract address.
    function computeCreate2Address(bytes memory _code, bytes32 _salt) public pure returns (address expected_) {
        bytes32 codeHash = keccak256(_code);
        expected_ = address(
            uint160(
                uint256(
                    keccak256(abi.encodePacked(bytes1(0xff), Preinstalls.DeterministicDeploymentProxy, _salt, codeHash))
                )
            )
        );
    }

    /// @notice Creates a deployment transaction via ConditionalDeployer.
    /// @dev The transaction calls ConditionalDeployer.deploy(salt, code) which performs
    ///      idempotent CREATE2 deployment via the DeterministicDeploymentProxy.
    /// @param _upgradeName Human-readable upgrade name (e.g., "Jovian").
    /// @param _name Human-readable name for the contract being deployed.
    /// @param _artifactPath Forge artifact path (e.g., "MyContract.sol:MyContract").
    /// @param _salt CREATE2 salt for address computation.
    /// @param _gasLimit Gas limit for the deployment transaction.
    /// @return txn_ The constructed deployment transaction.
    function createDeploymentTxn(
        string memory _upgradeName,
        string memory _name,
        string memory _artifactPath,
        bytes32 _salt,
        uint64 _gasLimit
    )
        public
        view
        returns (NetworkUpgradeTxns.NetworkUpgradeTxn memory txn_)
    {
        return createDeploymentTxnWithArgs(_upgradeName, _name, _artifactPath, "", _salt, _gasLimit);
    }

    /// @notice Creates a deployment transaction via ConditionalDeployer with constructor arguments.
    /// @dev The transaction calls ConditionalDeployer.deploy(salt, code) which performs
    ///      idempotent CREATE2 deployment via the DeterministicDeploymentProxy.
    /// @param _upgradeName Human-readable upgrade name (e.g., "Jovian").
    /// @param _name Human-readable name for the contract being deployed.
    /// @param _artifactPath Forge artifact path (e.g., "MyContract.sol:MyContract").
    /// @param _args ABI-encoded constructor arguments.
    /// @param _salt CREATE2 salt for address computation.
    /// @param _gasLimit Gas limit for the deployment transaction.
    /// @return txn_ The constructed deployment transaction.
    function createDeploymentTxnWithArgs(
        string memory _upgradeName,
        string memory _name,
        string memory _artifactPath,
        bytes memory _args,
        bytes32 _salt,
        uint64 _gasLimit
    )
        public
        view
        returns (NetworkUpgradeTxns.NetworkUpgradeTxn memory txn_)
    {
        bytes memory code = abi.encodePacked(vm.getCode(_artifactPath), _args);
        txn_ = NetworkUpgradeTxns.NetworkUpgradeTxn({
            sourceHash: NetworkUpgradeTxns.sourceHash(string.concat(_upgradeName, ": Deploy ", _name, " Implementation")),
            from: Constants.DEPOSITOR_ACCOUNT,
            to: Predeploys.CONDITIONAL_DEPLOYER,
            mint: 0,
            value: 0,
            gas: _gasLimit,
            isSystemTransaction: false,
            data: abi.encodeCall(ConditionalDeployer.deploy, (_salt, code))
        });
    }

    /// @notice Creates an upgrade transaction for a proxy contract.
    /// @dev The transaction calls IProxy(proxy).upgradeTo(implementation).
    ///      For the ProxyAdmin upgrade, the sender must be address(0) to use the
    ///      zero-address upgrade path in the Proxy.sol implementation.
    /// @param _upgradeName Human-readable upgrade name (e.g., "Jovian").
    /// @param _name Human-readable name for the contract being upgraded.
    /// @param _proxy Address of the proxy contract.
    /// @param _implementation Address of the new implementation.
    /// @param _gasLimit Gas limit for the upgrade transaction.
    /// @return txn_ The constructed upgrade transaction.
    function createUpgradeTxn(
        string memory _upgradeName,
        string memory _name,
        address _proxy,
        address _implementation,
        uint64 _gasLimit
    )
        public
        pure
        returns (NetworkUpgradeTxns.NetworkUpgradeTxn memory txn_)
    {
        txn_ = NetworkUpgradeTxns.NetworkUpgradeTxn({
            sourceHash: NetworkUpgradeTxns.sourceHash(string.concat(_upgradeName, ": Upgrade ", _name, " Implementation")),
            from: address(0),
            to: _proxy,
            mint: 0,
            value: 0,
            gas: _gasLimit,
            isSystemTransaction: false,
            data: abi.encodeCall(IProxy.upgradeTo, (_implementation))
        });
    }
}

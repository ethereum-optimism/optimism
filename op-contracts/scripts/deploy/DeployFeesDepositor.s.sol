// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";

import { IFeesDepositor } from "interfaces/L1/IFeesDepositor.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

/// @title DeployFeesDepositor
/// @notice Script used to deploy and initialize the FeesDepositor contract.
contract DeployFeesDepositor is Script {
    bytes32 internal _salt = DeployUtils.DEFAULT_SALT;

    address deployer;

    /// @notice Deploys and initializes the FeesDepositor contract.
    /// @param _proxyAdmin The address that will be the admin of the proxy.
    /// @param _minDepositAmount The threshold at which fees are deposited.
    /// @param _l2Recipient The L2 recipient of the fees.
    /// @param _messenger The L1CrossDomainMessenger contract address.
    /// @param _gasLimit The gas limit for the deposit transaction.
    function run(
        address _proxyAdmin,
        uint96 _minDepositAmount,
        address _l2Recipient,
        address _messenger,
        uint32 _gasLimit
    )
        public
        returns (IFeesDepositor, IProxy)
    {
        deployer = msg.sender;

        assertValidInput(_proxyAdmin, _l2Recipient, _messenger, _minDepositAmount, _gasLimit);

        // Deploy the implementation.
        IFeesDepositor impl = deployImplementation();

        // Deploy the proxy.
        IProxy proxy = deployProxy();

        // Initialize the proxy.
        initializeProxy(proxy, impl, _minDepositAmount, _l2Recipient, _messenger, _gasLimit);

        // Transfer the ownership of the proxy to the final proxy admin.
        transferToFinalProxyAdmin(_proxyAdmin, proxy);

        // Log the results.
        logResults(impl, proxy);

        return (impl, proxy);
    }

    /// @notice Deploys the FeesDepositor implementation contract.
    function deployImplementation() internal returns (IFeesDepositor) {
        return IFeesDepositor(
            DeployUtils.createDeterministic({
                _name: "FeesDepositor",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IFeesDepositor.__constructor__, ())),
                _salt: _salt
            })
        );
    }

    /// @notice Deploys the Proxy contract for FeesDepositor.
    function deployProxy() internal returns (IProxy) {
        return IProxy(
            DeployUtils.createDeterministic({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (deployer))),
                _salt: _salt
            })
        );
    }

    /// @notice Initializes the FeesDepositor proxy contract.
    /// @param _feesDepositorProxy The address of the FeesDepositor proxy.
    /// @param _feesDepositorImpl The address of the FeesDepositor implementation.
    /// @param _minDepositAmount The threshold at which fees are deposited.
    /// @param _l2Recipient The L2 recipient of the fees.
    /// @param _messenger The L1CrossDomainMessenger contract address.
    /// @param _gasLimit The gas limit for the deposit transaction.
    function initializeProxy(
        IProxy _feesDepositorProxy,
        IFeesDepositor _feesDepositorImpl,
        uint96 _minDepositAmount,
        address _l2Recipient,
        address _messenger,
        uint32 _gasLimit
    )
        internal
    {
        bytes memory initData = abi.encodeCall(
            IFeesDepositor.initialize, (_minDepositAmount, _l2Recipient, IL1CrossDomainMessenger(_messenger), _gasLimit)
        );

        vm.broadcast(deployer);
        IProxy(_feesDepositorProxy).upgradeToAndCall({ _implementation: address(_feesDepositorImpl), _data: initData });
    }

    /// @notice Transfers the ownership of the proxy to the final proxy admin.
    /// @param _proxyAdmin The address that will be the admin of the proxy.
    function transferToFinalProxyAdmin(address _proxyAdmin, IProxy _feesDepositorProxy) internal {
        vm.broadcast(deployer);
        _feesDepositorProxy.changeAdmin(_proxyAdmin);
    }

    /// @notice Validates the input parameters.
    /// @param _proxyAdmin The address that will be the admin of the proxy.
    /// @param _l2Recipient The L2 recipient of the fees.
    /// @param _messenger The L1CrossDomainMessenger contract address.
    /// @param _minDepositAmount The threshold at which fees are deposited.
    /// @param _gasLimit The gas limit for the deposit transaction.
    function assertValidInput(
        address _proxyAdmin,
        address _l2Recipient,
        address _messenger,
        uint96 _minDepositAmount,
        uint32 _gasLimit
    )
        internal
        pure
    {
        require(_proxyAdmin != address(0), "DeployFeesDepositor: proxyAdmin cannot be zero address");
        require(_l2Recipient != address(0), "DeployFeesDepositor: l2Recipient cannot be zero address");
        require(_messenger != address(0), "DeployFeesDepositor: messenger cannot be zero address");
        require(_minDepositAmount > 0, "DeployFeesDepositor: minDepositAmount must be greater than zero");
        require(_gasLimit > 0, "DeployFeesDepositor: gasLimit must be greater than zero");
    }

    /// @notice Logs the deployment results.
    /// @param _feesDepositorImpl The deployed FeesDepositor implementation address.
    /// @param _feesDepositorProxy The deployed FeesDepositor proxy address.
    function logResults(IFeesDepositor _feesDepositorImpl, IProxy _feesDepositorProxy) internal pure {
        console.log("=== FeesDepositor Deployment ===");
        console.log("Implementation:", address(_feesDepositorImpl));
        console.log("Proxy:", address(_feesDepositorProxy));
        console.log("================================");
    }
}

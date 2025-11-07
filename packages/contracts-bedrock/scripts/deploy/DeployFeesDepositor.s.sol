// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";

import { FeesDepositor } from "src/L1/FeesDepositor.sol";
import { IFeesDepositor } from "interfaces/L1/IFeesDepositor.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

/// @title DeployFeesDepositor
/// @notice Script used to deploy and initialize the FeesDepositor contract.
contract DeployFeesDepositor is Script {
    /// @notice Output addresses from deployment.
    struct Output {
        /// @notice The deployed FeesDepositor implementation address.
        address feesDepositorImpl;
        /// @notice The deployed FeesDepositor proxy address.
        address feesDepositorProxy;
    }

    bytes32 internal _salt = DeployUtils.DEFAULT_SALT;

    address deployer;

    /// @notice Deploys and initializes the FeesDepositor contract.
    /// @param _proxyAdmin The address that will be the admin of the proxy.
    /// @param _minDepositAmount The threshold at which fees are deposited.
    /// @param _l2Recipient The L2 recipient of the fees.
    /// @param _messenger The L1CrossDomainMessenger contract address.
    /// @param _gasLimit The gas limit for the deposit transaction.
    /// @return output_ The deployment output addresses.
    function run(
        address _proxyAdmin,
        uint96 _minDepositAmount,
        address _l2Recipient,
        address _messenger,
        uint32 _gasLimit
    )
        public
        returns (Output memory output_)
    {
        deployer = msg.sender;

        assertValidInput(_proxyAdmin, _l2Recipient, _messenger, _minDepositAmount, _gasLimit);

        // Deploy the implementation.
        deployImplementation(output_);

        // Deploy the proxy.
        deployProxy(_proxyAdmin, output_);

        // Initialize the proxy.
        initializeProxy(_minDepositAmount, _l2Recipient, _messenger, _gasLimit, output_);

        // Transfer the ownership of the proxy to the final proxy.
        transferToFinalProxyAdmin(_proxyAdmin, output_);

        // Log the results.
        logResults(output_);
    }

    /// @notice Deploys the FeesDepositor implementation contract.
    /// @param _output The output struct to populate.
    function deployImplementation(Output memory _output) internal {
        FeesDepositor impl = FeesDepositor(
            DeployUtils.createDeterministic({
                _name: "FeesDepositor",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IFeesDepositor.__constructor__, ())),
                _salt: _salt
            })
        );

        vm.label(address(impl), "FeesDepositorImpl");
        _output.feesDepositorImpl = address(impl);
    }

    /// @notice Deploys the Proxy contract for FeesDepositor.
    /// @param _proxyAdmin The address that will be the admin of the proxy.
    /// @param _output The output struct to populate.
    function deployProxy(address _proxyAdmin, Output memory _output) internal {
        IProxy proxy = IProxy(
            DeployUtils.createDeterministic({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (deployer))),
                _salt: _salt
            })
        );

        vm.label(address(proxy), "FeesDepositorProxy");
        _output.feesDepositorProxy = address(proxy);
    }

    /// @notice Initializes the FeesDepositor proxy contract.
    /// @param _minDepositAmount The threshold at which fees are deposited.
    /// @param _l2Recipient The L2 recipient of the fees.
    /// @param _messenger The L1CrossDomainMessenger contract address.
    /// @param _gasLimit The gas limit for the deposit transaction.
    /// @param _output The deployment output addresses.
    function initializeProxy(
        uint96 _minDepositAmount,
        address _l2Recipient,
        address _messenger,
        uint32 _gasLimit,
        Output memory _output
    )
        internal
    {
        bytes memory initData = abi.encodeCall(
            FeesDepositor.initialize, (_minDepositAmount, _l2Recipient, IL1CrossDomainMessenger(_messenger), _gasLimit)
        );

        console.log("Proxy admin:", deployer);
        console.logBytes32(
            vm.load(address(_output.feesDepositorProxy), bytes32(uint256(keccak256("eip1967.proxy.admin")) - 1))
        );
        vm.broadcast(deployer);
        IProxy(payable(_output.feesDepositorProxy))
            .upgradeToAndCall({ _implementation: _output.feesDepositorImpl, _data: initData });
    }

    /// @notice Transfers the ownership of the proxy to the final proxy.
    /// @param _proxyAdmin The address that will be the admin of the proxy.
    function transferToFinalProxyAdmin(address _proxyAdmin, Output memory _output) internal {
        vm.broadcast(deployer);
        IProxy(payable(_output.feesDepositorProxy)).changeAdmin(_proxyAdmin);
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
    /// @param _output The deployment output addresses.
    function logResults(Output memory _output) internal view {
        console.log("=== FeesDepositor Deployment ===");
        console.log("Implementation:", _output.feesDepositorImpl);
        console.log("Proxy:", _output.feesDepositorProxy);
        console.log("================================");
    }
}


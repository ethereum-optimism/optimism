// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Script } from "forge-std/Script.sol";

// Contracts
import { StorageSetter } from "src/universal/StorageSetter.sol";
import { MockComplianceModule } from "scripts/deploy/mocks/MockComplianceModule.sol";
import { MockOptimismPortal2 } from "scripts/deploy/mocks/MockOptimismPortal2.sol";
import { MockPolicy } from "scripts/deploy/mocks/MockPolicy.sol";

// Interfaces
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IBridgeHook } from "interfaces/L1/IBridgeHook.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IStorageSetter } from "interfaces/universal/IStorageSetter.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

/// @title DeployComplianceMocks
/// @notice Deploys the devnet compliance mocks and points a live OptimismPortal's bridge hook
///         at the module.
/// @dev    DEVNET ONLY.
///
///         The Portal already stores a bridge hook and its initializer already sets one from a
///         non-zero argument, but the initializer is a reinitializer whose version the live
///         proxy has already consumed, and that version is immutable in the implementation. So
///         the hook is wired by re-running initialize, which means clearing the proxy's
///         initialized byte first. That is the same sequence OPCM uses in
///         OPContractsManagerUtils.upgrade, reduced to the three calls needed here.
///
///         The Portal is upgraded to the implementation it already had. No new Portal
///         implementation exists anywhere in this change; the upgrade is only the vehicle for
///         re-running the initializer.
///
///         Every value initialize is given is read back off the Portal first, so the call is a
///         no-op for everything except the hook.
contract DeployComplianceMocks is Script {
    struct Input {
        IOptimismPortal2 portal;
        IProxyAdmin proxyAdmin;
        /// @notice Owner of both mocks. Must be the broadcaster, because the script wires them.
        address owner;
        /// @notice Address allowed to clear and revoke verdicts. The off-chain Verdict Writer.
        address screener;
        /// @notice Whether to also replace the Portal's implementation with the mock, which
        ///         screens deposits and withdrawals through the hook. Leave false once the real
        ///         Portal's hook paths have landed.
        bool useMockPortal;
    }

    struct Output {
        MockPolicy policy;
        MockComplianceModule module;
        /// @notice The implementation the Portal was left on. The original unless useMockPortal.
        address portalImpl;
    }

    /// @notice The Portal's configuration, read before the implementation is swapped out.
    struct PortalConfig {
        ISystemConfig systemConfig;
        IAnchorStateRegistry anchorStateRegistry;
        IETHLockbox ethLockbox;
        uint256 proofMaturityDelaySeconds;
        uint256 minProofMaturityDelaySeconds;
        uint256 maxProofMaturityDelaySeconds;
        address implementation;
    }

    /// @notice Slot holding the Initializable `_initialized` byte, at offset 0.
    bytes32 internal constant INITIALIZED_SLOT = bytes32(0);

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        // Must happen before the Portal is pointed at the StorageSetter, which has none of
        // these getters.
        PortalConfig memory config = readPortalConfig(_input);

        deployMocks(_input, output_);
        wireBridgeHook(_input, config, output_);

        assertValidOutput(_input, config, output_);
    }

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = run(input);
        return abi.encode(output);
    }

    /// @notice Reads the configuration that will be handed straight back to initialize.
    function readPortalConfig(Input memory _input) public view returns (PortalConfig memory config_) {
        config_ = PortalConfig({
            systemConfig: _input.portal.systemConfig(),
            anchorStateRegistry: _input.portal.anchorStateRegistry(),
            ethLockbox: _input.portal.ethLockbox(),
            proofMaturityDelaySeconds: _input.portal.proofMaturityDelaySeconds(),
            minProofMaturityDelaySeconds: _input.portal.minProofMaturityDelaySeconds(),
            maxProofMaturityDelaySeconds: _input.portal.maxProofMaturityDelaySeconds(),
            implementation: _input.proxyAdmin.getProxyImplementation(address(_input.portal))
        });
    }

    /// @notice Deploys the policy and the module and points each at the other.
    function deployMocks(Input memory _input, Output memory _output) public {
        vm.broadcast(msg.sender);
        _output.policy = new MockPolicy(_input.owner, _input.screener);

        vm.broadcast(msg.sender);
        _output.module = new MockComplianceModule(_input.portal, _input.owner);

        vm.broadcast(msg.sender);
        _output.policy.setModule(IComplianceModule(address(_output.module)));

        vm.broadcast(msg.sender);
        _output.module.setPolicy(IPolicy(address(_output.policy)));
    }

    /// @notice Clears the Portal's initialized byte and re-runs its initializer with the module
    ///         as the bridge hook.
    function wireBridgeHook(Input memory _input, PortalConfig memory _config, Output memory _output) public {
        address portal = address(_input.portal);

        _output.portalImpl = _config.implementation;
        if (_input.useMockPortal) {
            // Same constructor arguments as the implementation being replaced, so the proof
            // maturity bounds the Portal was deployed with are preserved. Read into locals
            // first: vm.broadcast applies to the next call, and a staticcall would consume it.
            uint256 minDelay = _config.minProofMaturityDelaySeconds;
            uint256 maxDelay = _config.maxProofMaturityDelaySeconds;

            vm.broadcast(msg.sender);
            _output.portalImpl = address(new MockOptimismPortal2(minDelay, maxDelay));
        }

        vm.broadcast(msg.sender);
        StorageSetter storageSetter = new StorageSetter();

        vm.broadcast(msg.sender);
        _input.proxyAdmin.upgrade(payable(portal), address(storageSetter));

        // Clear only the `_initialized` byte. The adjacent `_initializing` byte must be
        // preserved, and must already be zero, or the proxy is mid-initialization.
        bytes32 current = IStorageSetter(portal).getBytes32(INITIALIZED_SLOT);
        require((uint256(current) >> 8) & 0xff == 0, "DeployComplianceMocks: portal is initializing");

        vm.broadcast(msg.sender);
        IStorageSetter(portal).setBytes32(INITIALIZED_SLOT, bytes32(uint256(current) & ~uint256(0xff)));

        vm.broadcast(msg.sender);
        _input.proxyAdmin
            .upgradeAndCall(
                payable(portal),
                _output.portalImpl,
                abi.encodeCall(
                    IOptimismPortal2.initialize,
                    (
                        _config.systemConfig,
                        _config.anchorStateRegistry,
                        _config.ethLockbox,
                        _config.proofMaturityDelaySeconds,
                        IBridgeHook(address(_output.module))
                    )
                )
            );
    }

    function assertValidInput(Input memory _input) public view {
        require(address(_input.portal) != address(0), "DeployComplianceMocks: portal not set");
        require(address(_input.proxyAdmin) != address(0), "DeployComplianceMocks: proxyAdmin not set");
        require(_input.screener != address(0), "DeployComplianceMocks: screener not set");

        // The script wires both mocks, so the broadcaster has to be able to.
        require(_input.owner == msg.sender, "DeployComplianceMocks: owner must be the broadcaster");

        // Every call below is onlyOwner on the ProxyAdmin.
        require(_input.proxyAdmin.owner() == msg.sender, "DeployComplianceMocks: not the ProxyAdmin owner");

        // Refuses a second run rather than silently replacing a hook that may already hold
        // value. Removing the existing hook first is a deliberate decision, not a default.
        require(address(_input.portal.bridgeHook()) == address(0), "DeployComplianceMocks: hook already set");
    }

    function assertValidOutput(Input memory _input, PortalConfig memory _config, Output memory _output) public view {
        require(address(_input.portal.bridgeHook()) == address(_output.module), "DeployComplianceMocks: hook not set");
        require(address(_output.module.policy()) == address(_output.policy), "DeployComplianceMocks: policy not set");
        require(address(_output.module.portal()) == address(_input.portal), "DeployComplianceMocks: wrong portal");
        require(address(_output.policy.module()) == address(_output.module), "DeployComplianceMocks: module not set");

        // The Portal must be back on the implementation it started with, and everything
        // initialize was handed must be unchanged.
        require(
            _input.proxyAdmin.getProxyImplementation(address(_input.portal)) == _output.portalImpl,
            "DeployComplianceMocks: wrong implementation"
        );
        require(
            _input.useMockPortal || _output.portalImpl == _config.implementation,
            "DeployComplianceMocks: implementation changed"
        );
        require(_input.portal.systemConfig() == _config.systemConfig, "DeployComplianceMocks: systemConfig changed");
        require(
            _input.portal.anchorStateRegistry() == _config.anchorStateRegistry,
            "DeployComplianceMocks: anchorStateRegistry changed"
        );
        require(_input.portal.ethLockbox() == _config.ethLockbox, "DeployComplianceMocks: ethLockbox changed");
        require(
            _input.portal.proofMaturityDelaySeconds() == _config.proofMaturityDelaySeconds,
            "DeployComplianceMocks: proofMaturityDelaySeconds changed"
        );
    }
}

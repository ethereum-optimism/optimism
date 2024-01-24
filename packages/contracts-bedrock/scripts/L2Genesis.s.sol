// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

import { L2StandardBridge } from "src/L2/L2StandardBridge.sol";
import { L2CrossDomainMessenger } from "src/L2/L2CrossDomainMessenger.sol";
import { SequencerFeeVault } from "src/L2/SequencerFeeVault.sol";
import { FeeVault } from "src/universal/FeeVault.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { GovernanceToken } from "src/governance/GovernanceToken.sol";

interface IInitializable {
    function initialize() external;
}

// @title
contract L2Genesis is Script {
    uint256 constant PROXY_COUNT = 2048;
    uint256 constant PRECOMPILE_COUNT = 256;

    /// @notice The storage slot that holds the address of a proxy implementation.
    /// @dev `bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)`
    bytes32 internal constant PROXY_IMPLEMENTATION_ADDRESS =
        0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

    /// @notice The storage slot that holds the address of the owner.
    /// @dev `bytes32(uint256(keccak256('eip1967.proxy.admin')) - 1)`
    bytes32 internal constant PROXY_ADMIN_ADDRESS = 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103;

    string outfile;

    function setUp() external {
        outfile = string.concat(vm.projectRoot(), "/genesis.json");
    }

    function run() public {
        _setPrecompiles();
        _setProxies();
        _setImplementations();

        vm.dumpState(outfile);
    }

    /// @notice Give all of the precompiles 1 wei so that they are
    ///         not considered empty accounts.
    function _setPrecompiles() internal {
        for (uint256 i; i < PRECOMPILE_COUNT; i++) {
            vm.deal(address(uint160(i)), 1);
        }
    }

    /// @dev Set up the accounts that correspond to the predeploys.
    ///      The Proxy bytecode should be set. All proxied predeploys should have
    ///      the 1967 admin slot set to the ProxyAdmin predeploy. All defined predeploys
    ///      should have their implementations set.
    function _setProxies() internal {
        bytes memory code = vm.getDeployedCode("Proxy.sol:Proxy");
        uint160 prefix = uint160(0x420) << 148;
        for (uint256 i = 0; i < PROXY_COUNT; i++) {
            address addr = address(prefix | uint160(i));
            if (_notProxied(addr)) {
                continue;
            }

            vm.etch(addr, code);
            vm.store(addr, PROXY_ADMIN_ADDRESS, bytes32(uint256(uint160(Predeploys.PROXY_ADMIN))));

            if (_isDefinedPredeploy(addr)) {
                address implementation = _predeployToCodeNamespace(addr);
                vm.store(addr, PROXY_IMPLEMENTATION_ADDRESS, bytes32(uint256(uint160(implementation))));
            }
        }
    }

    /// @dev
    /// 2 options
    ///  - use getDeployedCode
    ///  - use new
    /// need to ensure that storage is correct
    /// need assert no immutables sort of check
    function _setImplementations() internal {
        _setLegacyMessagePasser();
        _setDeployerWhitelist();
        _setLegacyERC20ETH();
        _setWETH9();
        _setL2CrossDomainMessenger();
        _setL2StandardBridge();
        _setSequencerFeeVault();
        _setOptimismMintableERC20Factory();
        _setL1BlockNumber();
        _setGasPriceOracle();
        _setGovernanceToken();
        _setL1Block();
    }

    function _setLegacyMessagePasser() internal {
        address impl = _predeployToCodeNamespace(Predeploys.LEGACY_MESSAGE_PASSER);
        vm.etch(impl, vm.getDeployedCode("LegacyMessagePasser.sol:LegacyMessagePasser"));
    }

    function _setDeployerWhitelist() internal {
        address impl = _predeployToCodeNamespace(Predeploys.DEPLOYER_WHITELIST);
        vm.etch(impl, vm.getDeployedCode("DeployerWhitelist.sol:DeployerWhitelist"));
    }

    function _setLegacyERC20ETH() internal {
        // TODO: this is ignored in Go code?
    }

    function _setWETH9() internal {
        vm.etch(Predeploys.WETH9, vm.getDeployedCode("WETH9.sol:WETH9"));
    }

    function _setL2StandardBridge() internal {
        address l1CrossDomainMessenger = address(0);
        address impl = _predeployToCodeNamespace(Predeploys.L2_STANDARD_BRIDGE);

        L2StandardBridge bridge = new L2StandardBridge(payable(l1CrossDomainMessenger));
        vm.etch(impl, address(bridge).code);

        IInitializable(impl).initialize();
        IInitializable(Predeploys.L2_STANDARD_BRIDGE).initialize();

        vm.etch(address(bridge), hex"");
        vm.resetNonce(address(bridge));
    }

    /// @notice
    function _setL2CrossDomainMessenger() internal {
        // TODO: wrong argument
        address l1CrossDomainMessenger = address(0);
        address impl = _predeployToCodeNamespace(Predeploys.L2_CROSS_DOMAIN_MESSENGER);

        L2CrossDomainMessenger messenger = new L2CrossDomainMessenger(l1CrossDomainMessenger);

        vm.etch(impl, address(messenger).code);
        IInitializable(impl).initialize();
        IInitializable(Predeploys.L2_CROSS_DOMAIN_MESSENGER).initialize();

        // TODO: upstream filtering out of empty accounts?
        vm.etch(address(messenger), hex"");
        vm.resetNonce(address(messenger));
    }

    function _setSequencerFeeVault() internal {
        SequencerFeeVault vault = new SequencerFeeVault({
            _recipient: address(0),
            _minWithdrawalAmount: 0,
            _withdrawalNetwork: FeeVault.WithdrawalNetwork.L2
        });

        vm.etch(_predeployToCodeNamespace(Predeploys.SEQUENCER_FEE_WALLET), address(vault).code);

        vm.etch(address(vault), hex"");
        vm.resetNonce(address(vault));
    }

    function _setOptimismMintableERC20Factory() internal {
        OptimismMintableERC20Factory factory = new OptimismMintableERC20Factory({
            _bridge: Predeploys.L2_STANDARD_BRIDGE
        });

        vm.etch(_predeployToCodeNamespace(Predeploys.SEQUENCER_FEE_WALLET), address(factory).code);

        vm.etch(address(factory), hex"");
        vm.resetNonce(address(factory));
    }

    function _setL1BlockNumber() internal {
        vm.etch(
            _predeployToCodeNamespace(Predeploys.L1_BLOCK_NUMBER),
            vm.getDeployedCode("L1BlockNumber.sol:L1BlockNumber")
        );
    }

    function _setGasPriceOracle() internal {
        vm.etch(
            _predeployToCodeNamespace(Predeploys.GAS_PRICE_ORACLE),
            vm.getDeployedCode("GasPriceOracle.sol:GasPriceOracle")
        );
    }

    // TODO: needs conditional logic to not include based on config
    function _setGovernanceToken() internal {
        GovernanceToken token = new GovernanceToken();
        vm.etch(Predeploys.GOVERNANCE_TOKEN, address(token).code);

        vm.etch(address(token), hex"");
        vm.resetNonce(address(token));
    }

    function _setL1Block() internal {
        vm.etch(
            _predeployToCodeNamespace(Predeploys.L1_BLOCK_ATTRIBUTES),
            vm.getDeployedCode("L1Block.sol:L1Block")
        );
    }

    /// @dev
    function _setPreinstalls() internal {
        //
    }

    /// @dev Function to compute the expected address of the predeploy implementation
    ///      in the genesis state.
    function _predeployToCodeNamespace(address _addr) internal pure returns (address) {
        return address(
            uint160(uint256(uint160(_addr)) & 0xffff | uint256(uint160(0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000)))
        );
    }

    /// @dev Returns true if the address is a predeploy.
    function _isDefinedPredeploy(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.L2_TO_L1_MESSAGE_PASSER || _addr == Predeploys.L2_CROSS_DOMAIN_MESSENGER
            || _addr == Predeploys.L2_STANDARD_BRIDGE || _addr == Predeploys.L2_ERC721_BRIDGE
            || _addr == Predeploys.SEQUENCER_FEE_WALLET || _addr == Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY
            || _addr == Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY || _addr == Predeploys.L1_BLOCK_ATTRIBUTES
            || _addr == Predeploys.GAS_PRICE_ORACLE || _addr == Predeploys.DEPLOYER_WHITELIST || _addr == Predeploys.WETH9
            || _addr == Predeploys.L1_BLOCK_NUMBER || _addr == Predeploys.LEGACY_MESSAGE_PASSER
            || _addr == Predeploys.PROXY_ADMIN || _addr == Predeploys.BASE_FEE_VAULT || _addr == Predeploys.L1_FEE_VAULT
            || _addr == Predeploys.GOVERNANCE_TOKEN || _addr == Predeploys.SCHEMA_REGISTRY || _addr == Predeploys.EAS;
    }

    /// @dev Returns true if the adress is not proxied.
    function _notProxied(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.GOVERNANCE_TOKEN || _addr == Predeploys.WETH9;
    }
}

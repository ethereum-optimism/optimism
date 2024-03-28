// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";
import { Deployer } from "scripts/Deployer.sol";

import { Config } from "scripts/Config.sol";
import { Artifacts } from "scripts/Artifacts.s.sol";
import { DeployConfig } from "scripts/DeployConfig.s.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { L2CrossDomainMessenger } from "src/L2/L2CrossDomainMessenger.sol";
import { L2StandardBridge } from "src/L2/L2StandardBridge.sol";
import { L2ERC721Bridge } from "src/L2/L2ERC721Bridge.sol";
import { SequencerFeeVault } from "src/L2/SequencerFeeVault.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { OptimismMintableERC721Factory } from "src/universal/OptimismMintableERC721Factory.sol";
import { BaseFeeVault } from "src/L2/BaseFeeVault.sol";
import { L1FeeVault } from "src/L2/L1FeeVault.sol";
import { GovernanceToken } from "src/governance/GovernanceToken.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { FeeVault } from "src/universal/FeeVault.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

interface IInitializable {
    function initialize(address _addr) external;
}

/// @title L2Genesis
/// @notice Generates the genesis state for the L2 network.
///         The following safety invariants are used when setting state:
///         1. `vm.getDeployedBytecode` can only be used with `vm.etch` when there are no side
///         effects in the constructor and no immutables in the bytecode.
///         2. A contract must be deployed using the `new` syntax if there are immutables in the code.
///         Any other side effects from the init code besides setting the immutables must be cleaned up afterwards.
contract L2Genesis is Deployer {
    uint256 constant public PREDEPLOY_COUNT = 2048;
    uint256 constant public PRECOMPILE_COUNT = 256;

    uint80 internal constant DEV_ACCOUNT_FUND_AMT = 10_000 ether;
    /// @notice Default Anvil dev accounts. Only funded if `cfg.fundDevAccounts == true`.
    address[10] internal devAccounts = [
        0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266,
        0x70997970C51812dc3A010C7d01b50e0d17dc79C8,
        0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC,
        0x90F79bf6EB2c4f870365E785982E1f101E93b906,
        0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65,
        0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc,
        0x976EA74026E726554dB657fA54763abd0C3a0aa9,
        0x14dC79964da2C08b23698B3D3cc7Ca32193d9955,
        0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f,
        0xa0Ee7A142d267C1f36714E4a8F75612F20a79720
    ];


    mapping(address => string) internal names;

    function name() public pure override returns (string memory) {
        return "L2Genesis";
    }

    /// @dev Reads the deploy config, sets `outfile` which is where the `vm.dumpState` will be saved to, and
    ///      loads in the addresses for the L1 contract deployments.
    function setUp() public override {
        super.setUp();

        // TODO: modularize this setNames into own contract
        _setNames();
    }

    /// @dev Creates a mapping of predeploy addresses to their names. This needs to be updated
    ///      any time there is a new predeploy added.
    function _setNames() internal {
        names[Predeploys.L2_TO_L1_MESSAGE_PASSER] = "L2ToL1MessagePasser";
        names[Predeploys.L2_CROSS_DOMAIN_MESSENGER] = "L2CrossDomainMessenger";
        names[Predeploys.L2_STANDARD_BRIDGE] = "L2StandardBridge";
        names[Predeploys.L2_ERC721_BRIDGE] = "L2ERC721Bridge";
        names[Predeploys.SEQUENCER_FEE_WALLET] = "SequencerFeeVault";
        names[Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY] = "OptimismMintableERC20Factory";
        names[Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY] = "OptimismMintableERC721Factory";
        names[Predeploys.L1_BLOCK_ATTRIBUTES] = "L1Block";
        names[Predeploys.GAS_PRICE_ORACLE] = "GasPriceOracle";
        names[Predeploys.L1_MESSAGE_SENDER] = "L1MessageSender";
        names[Predeploys.DEPLOYER_WHITELIST] = "DeployerWhitelist";
        names[Predeploys.WETH9] = "WETH9";
        names[Predeploys.LEGACY_ERC20_ETH] = "LegacyERC20ETH";
        names[Predeploys.L1_BLOCK_NUMBER] = "L1BlockNumber";
        names[Predeploys.LEGACY_MESSAGE_PASSER] = "LegacyMessagePasser";
        names[Predeploys.PROXY_ADMIN] = "ProxyAdmin";
        names[Predeploys.BASE_FEE_VAULT] = "BaseFeeVault";
        names[Predeploys.L1_FEE_VAULT] = "L1FeeVault";
        names[Predeploys.GOVERNANCE_TOKEN] = "GovernanceToken";
        names[Predeploys.SCHEMA_REGISTRY] = "SchemaRegistry";
        names[Predeploys.EAS] = "EAS";
    }

    /// @dev Sets the precompiles, proxies, and the implementation accounts to be `vm.dumpState`
    ///      to generate a L2 genesis alloc.
    /// @notice The alloc object is sorted numerically by address.
    function run() public {
        dealEthToPrecompiles();
        setPredeployProxies();
        setPredeployImplementations();

        if (cfg.fundDevAccounts()) {
            fundDevAccounts();
        }

        writeStateDump();
    }

    /// @notice Give all of the precompiles 1 wei
    function dealEthToPrecompiles() internal {
        for (uint256 i; i < PRECOMPILE_COUNT; i++) {
            vm.deal(address(uint160(i)), 1);
        }
    }

    /// @dev Set up the accounts that correspond to the predeploys.
    ///      The Proxy bytecode should be set. All proxied predeploys should have
    ///      the 1967 admin slot set to the ProxyAdmin predeploy. All defined predeploys
    ///      should have their implementations set.
    function setPredeployProxies() public {
        bytes memory code = vm.getDeployedCode("Proxy.sol:Proxy");
        uint160 prefix = uint160(0x420) << 148;

        console.log(
            "Setting proxy deployed bytecode for addresses in range %s through %s",
            address(prefix | uint160(0)),
            address(prefix | uint160(PREDEPLOY_COUNT - 1))
        );
        for (uint256 i = 0; i < PREDEPLOY_COUNT; i++) {
            address addr = address(prefix | uint160(i));
            if (_notProxied(addr)) {
                console.log("Skipping proxy at %s", addr);
                continue;
            }

            vm.etch(addr, code);
            EIP1967Helper.setAdmin(addr, Predeploys.PROXY_ADMIN);

            if (_isDefinedPredeploy(addr)) {
                address implementation = predeployToCodeNamespace(addr);
                console.log("Setting proxy %s implementation: %s", addr, implementation);
                EIP1967Helper.setImplementation(addr, implementation);
            }
        }
    }

    /// @dev Sets all the implementations for the predeploy proxies. For contracts without proxies,
    ///      sets the deployed bytecode at their expected predeploy address.
    ///      LEGACY_ERC20_ETH and L1_MESSAGE_SENDER are deprecated and are not set.
    function setPredeployImplementations() internal {
        setL2ToL1MessagePasser();
        setL2CrossDomainMessenger();
        setL2StandardBridge();
        setL2ERC721Bridge();
        setSequencerFeeVault();
        setOptimismMintableERC20Factory();
        setOptimismMintableERC721Factory();
        setL1Block();
        setGasPriceOracle();
        setDeployerWhitelist();
        setWETH9();
        setL1BlockNumber();
        setLegacyMessagePasser();
        setBaseFeeVault();
        setL1FeeVault();
        setGovernanceToken();
        setSchemaRegistry();
        setEAS();
    }

    function setL2ToL1MessagePasser() public {
        _setImplementationCode(Predeploys.L2_TO_L1_MESSAGE_PASSER);
    }

    /// @notice This predeploy is following the saftey invariant #1.
    function setL2CrossDomainMessenger() public {
        address impl = _setImplementationCode(Predeploys.L2_CROSS_DOMAIN_MESSENGER);

        L2CrossDomainMessenger(impl).initialize({
            _l1CrossDomainMessenger: L1CrossDomainMessenger(address(0))
        });

        L2CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).initialize({
            _l1CrossDomainMessenger: L1CrossDomainMessenger(mustGetAddress("L1CrossDomainMessengerProxy"))
        });
    }

    /// @notice This predeploy is following the saftey invariant #1.
    function setL2StandardBridge() public {
        address impl = _setImplementationCode(Predeploys.L2_STANDARD_BRIDGE);

        L2StandardBridge(payable(impl)).initialize({
            _otherBridge: L1StandardBridge(payable(address(0)))
        });

        L2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).initialize({
            _otherBridge: L1StandardBridge(mustGetAddress("L1StandardBridgeProxy"))
        });
    }

    /// @notice This predeploy is following the saftey invariant #1.
    function setL2ERC721Bridge() public {
        address impl = _setImplementationCode(Predeploys.L2_ERC721_BRIDGE);

        L2ERC721Bridge(impl).initialize({
            _l1ERC721Bridge: payable(address(0))
        });

        L2ERC721Bridge(Predeploys.L2_ERC721_BRIDGE).initialize({
            _l1ERC721Bridge: payable(mustGetAddress("L1ERC721BridgeProxy"))
        });
    }

    /// @notice This predeploy is following the saftey invariant #2,
    function setSequencerFeeVault() public {
        SequencerFeeVault vault = new SequencerFeeVault({
            _recipient: cfg.sequencerFeeVaultRecipient(),
            _minWithdrawalAmount: cfg.sequencerFeeVaultMinimumWithdrawalAmount(),
            _withdrawalNetwork: FeeVault.WithdrawalNetwork(cfg.sequencerFeeVaultWithdrawalNetwork())
        });

        address impl = predeployToCodeNamespace(Predeploys.SEQUENCER_FEE_WALLET);
        console.log("Setting %s implementation at: %s", "SequencerFeeVault", impl);
        vm.etch(impl, address(vault).code);

        /// Reset so its not included state dump
        vm.etch(address(vault), "");
        vm.resetNonce(address(vault));
    }

    /// @notice This predeploy is following the saftey invariant #1.
    function setOptimismMintableERC20Factory() public {
        address impl = _setImplementationCode(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY);

        OptimismMintableERC20Factory(impl).initialize({
            _bridge: address(0)
        });

        OptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY).initialize({
            _bridge: Predeploys.L2_STANDARD_BRIDGE
        });
    }

    /// @notice This predeploy is following the saftey invariant #2,
    function setOptimismMintableERC721Factory() public {
        OptimismMintableERC721Factory factory = new OptimismMintableERC721Factory({
            _bridge: Predeploys.L2_ERC721_BRIDGE,
            _remoteChainId: cfg.l1ChainID()
        });

        address impl = predeployToCodeNamespace(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY);
        console.log("Setting %s implementation at: %s", "OptimismMintableERC721Factory", impl);
        vm.etch(impl, address(factory).code);

        /// Reset so its not included state dump
        vm.etch(address(factory), "");
        vm.resetNonce(address(factory));
    }

    /// @notice This predeploy is following the saftey invariant #1.
    function setL1Block() public {
        _setImplementationCode(Predeploys.L1_BLOCK_ATTRIBUTES);
    }

    /// @notice This predeploy is following the saftey invariant #1.
    function setGasPriceOracle() public {
        _setImplementationCode(Predeploys.GAS_PRICE_ORACLE);
    }

    /// @notice This predeploy is following the saftey invariant #1.
    function setDeployerWhitelist() public {
        _setImplementationCode(Predeploys.DEPLOYER_WHITELIST);
    }

    /// @notice This predeploy is following the saftey invariant #1.
    ///         This contract is NOT proxied and the state that is set
    ///         in the constructor is set manually.
    function setWETH9() public {
        console.log("Setting %s implementation at: %s", "WETH9", Predeploys.WETH9);
        vm.etch(Predeploys.WETH9, vm.getDeployedCode("WETH9.sol:WETH9"));

        vm.store(
            Predeploys.WETH9,
            /// string public name
            hex"0000000000000000000000000000000000000000000000000000000000000000",
            /// "Wrapped Ether"
            hex"577261707065642045746865720000000000000000000000000000000000001a"
        );
        vm.store(
            Predeploys.WETH9,
            /// string public symbol
            hex"0000000000000000000000000000000000000000000000000000000000000001",
            /// "WETH"
            hex"5745544800000000000000000000000000000000000000000000000000000008"
        );
        vm.store(
            Predeploys.WETH9,
            // uint8 public decimals
            hex"0000000000000000000000000000000000000000000000000000000000000002",
            /// 18
            hex"0000000000000000000000000000000000000000000000000000000000000012"
        );
    }

    /// @notice This predeploy is following the saftey invariant #1.
    function setL1BlockNumber() public {
        _setImplementationCode(Predeploys.L1_BLOCK_NUMBER);
    }

    /// @notice This predeploy is following the saftey invariant #1.
    function setLegacyMessagePasser() public {
        _setImplementationCode(Predeploys.LEGACY_MESSAGE_PASSER);
    }

    /// @notice This predeploy is following the saftey invariant #2.
    function setBaseFeeVault() public {
        BaseFeeVault vault = new BaseFeeVault({
            _recipient: cfg.baseFeeVaultRecipient(),
            _minWithdrawalAmount: cfg.baseFeeVaultMinimumWithdrawalAmount(),
            _withdrawalNetwork: FeeVault.WithdrawalNetwork(cfg.baseFeeVaultWithdrawalNetwork())
        });

        address impl = predeployToCodeNamespace(Predeploys.BASE_FEE_VAULT);
        console.log("Setting %s implementation at: %s", "BaseFeeVault", impl);
        vm.etch(impl, address(vault).code);

        /// Reset so its not included state dump
        vm.etch(address(vault), "");
        vm.resetNonce(address(vault));
    }

    /// @notice This predeploy is following the saftey invariant #2.
    function setL1FeeVault() public {
        L1FeeVault vault = new L1FeeVault({
            _recipient: cfg.l1FeeVaultRecipient(),
            _minWithdrawalAmount: cfg.l1FeeVaultMinimumWithdrawalAmount(),
            _withdrawalNetwork: FeeVault.WithdrawalNetwork(cfg.l1FeeVaultWithdrawalNetwork())
        });

        address impl = predeployToCodeNamespace(Predeploys.L1_FEE_VAULT);
        console.log("Setting %s implementation at: %s", "L1FeeVault", impl);
        vm.etch(impl, address(vault).code);

        /// Reset so its not included state dump
        vm.etch(address(vault), "");
        vm.resetNonce(address(vault));
    }

    /// @notice This predeploy is following the saftey invariant #2.
    function setGovernanceToken() public {
        if (!cfg.enableGovernance()) {
            console.log("Governance not enabled, skipping setting governanace token");
            return;
        }

        GovernanceToken token = new GovernanceToken();
        console.log("Setting %s implementation at: %s", "GovernanceToken", Predeploys.GOVERNANCE_TOKEN);
        vm.etch(Predeploys.GOVERNANCE_TOKEN, address(token).code);

        bytes32 _nameSlot = hex"0000000000000000000000000000000000000000000000000000000000000003";
        bytes32 _symbolSlot = hex"0000000000000000000000000000000000000000000000000000000000000004";
        bytes32 _ownerSlot = hex"000000000000000000000000000000000000000000000000000000000000000a";

        vm.store(Predeploys.GOVERNANCE_TOKEN, _nameSlot, vm.load(address(token), _nameSlot));
        vm.store(Predeploys.GOVERNANCE_TOKEN, _symbolSlot, vm.load(address(token), _symbolSlot));
        vm.store(Predeploys.GOVERNANCE_TOKEN, _ownerSlot, bytes32(uint256(uint160(cfg.governanceTokenOwner()))));

        /// Reset so its not included state dump
        vm.etch(address(token), "");
        vm.resetNonce(address(token));
    }

    /// @notice This predeploy is following the saftey invariant #1.
    function setSchemaRegistry() public {
        _setImplementationCode(Predeploys.SCHEMA_REGISTRY);
    }

    /// @notice This predeploy is following the saftey invariant #2,
    ///         It uses low level create to deploy the contract due to the code
    ///         having immutables and being a different compiler version.
    function setEAS() public {
        string memory cname = names[Predeploys.EAS];
        address impl = predeployToCodeNamespace(Predeploys.EAS);
        bytes memory code = vm.getCode(string.concat(cname, ".sol:", cname));

        address eas;
        assembly {
            eas := create(0, add(code, 0x20), mload(code))
        }

        console.log("Setting %s implementation at: %s", cname, impl);
        vm.etch(impl, eas.code);

        /// Reset so its not included state dump
        vm.etch(address(eas), "");
        vm.resetNonce(address(eas));
    }

    /// @notice Returns true if the address is not proxied.
    function _notProxied(address _addr) internal pure returns (bool) {
        return _addr == Predeploys.GOVERNANCE_TOKEN || _addr == Predeploys.WETH9;
    }

    /// @notice Returns true if the address is a predeploy.
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

    /// @notice Function to compute the expected address of the predeploy implementation
    ///         in the genesis state.
    function predeployToCodeNamespace(address _addr) public pure returns (address) {
        return address(
            uint160(uint256(uint160(_addr)) & 0xffff | uint256(uint160(0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000)))
        );
    }

    /// @notice Sets the bytecode in state
    function _setImplementationCode(address _addr) internal returns (address) {
        string memory cname = names[_addr];
        address impl = predeployToCodeNamespace(_addr);
        console.log("Setting %s implementation at: %s", cname, impl);
        vm.etch(impl, vm.getDeployedCode(string.concat(cname, ".sol:", cname)));
        return impl;
    }

    /// @notice Writes the state dump to disk
    function writeStateDump() public {
        /// Reset so its not included state dump
        vm.etch(address(cfg), "");
        vm.etch(msg.sender, "");
        vm.resetNonce(msg.sender);
        vm.deal(msg.sender, 0);

        string memory path = Config.stateDumpPath();
        console.log("Writing state dump to: %s", path);
        vm.dumpState(path);
        sortJsonByKeys(path);
    }

    /// @notice Sorts the allocs by address
    function sortJsonByKeys(string memory _path) internal {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("cat <<< $(jq -S '.' ", _path, ") > ", _path);
        vm.ffi(commands);
    }

    /// @notice Funds the default dev accounts with ether
    function fundDevAccounts() internal {
        for (uint256 i; i < devAccounts.length; i++) {
            console.log("Funding dev account %s with %s ETH", devAccounts[i], DEV_ACCOUNT_FUND_AMT / 1e18);
            vm.deal(devAccounts[i], DEV_ACCOUNT_FUND_AMT);
        }
    }
}

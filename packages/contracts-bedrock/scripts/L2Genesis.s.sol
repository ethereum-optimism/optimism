// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";
import { Deployer } from "scripts/Deployer.sol";

import { Config } from "scripts/Config.sol";
import { Artifacts } from "scripts/Artifacts.s.sol";
import { DeployConfig } from "scripts/DeployConfig.s.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { L2CrossDomainMessenger } from "src/L2/L2CrossDomainMessenger.sol";
import { L1Block } from "src/L2/L1Block.sol";
import { GasPriceOracle } from "src/L2/GasPriceOracle.sol";
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

struct L1Dependencies {
    address payable l1CrossDomainMessengerProxy;
    address payable l1StandardBridgeProxy;
    address payable l1ERC721BridgeProxy;
}

/// @notice Enum representing different ways of outputting genesis allocs.
/// @custom:value DEFAULT_LATEST Represents only latest L2 allocs, written to output path.
/// @custom:value LOCAL_LATEST   Represents latest L2 allocs, not output anywhere, but kept in-process.
/// @custom:value LOCAL_ECOTONE  Represents Ecotone-upgrade L2 allocs, not output anywhere, but kept in-process.
/// @custom:value LOCAL_DELTA    Represents Delta-upgrade L2 allocs, not output anywhere, but kept in-process.
/// @custom:value OUTPUT_ALL     Represents creation of one L2 allocs file for every upgrade.
enum OutputMode {
    DEFAULT_LATEST,
    LOCAL_LATEST,
    LOCAL_ECOTONE,
    LOCAL_DELTA,
    OUTPUT_ALL
}

/// @title L2Genesis
/// @notice Generates the genesis state for the L2 network.
///         The following safety invariants are used when setting state:
///         1. `vm.getDeployedBytecode` can only be used with `vm.etch` when there are no side
///         effects in the constructor and no immutables in the bytecode.
///         2. A contract must be deployed using the `new` syntax if there are immutables in the code.
///         Any other side effects from the init code besides setting the immutables must be cleaned up afterwards.
contract L2Genesis is Deployer {
    uint256 public constant PRECOMPILE_COUNT = 256;

    uint80 internal constant DEV_ACCOUNT_FUND_AMT = 10_000 ether;

    /// @notice Default Anvil dev accounts. Only funded if `cfg.fundDevAccounts == true`.
    /// Also known as "test test test test test test test test test test test junk" mnemonic accounts,
    /// on path "m/44'/60'/0'/0/i" (where i is the account index).
    address[30] internal devAccounts = [
        0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266, // 0
        0x70997970C51812dc3A010C7d01b50e0d17dc79C8, // 1
        0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC, // 2
        0x90F79bf6EB2c4f870365E785982E1f101E93b906, // 3
        0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65, // 4
        0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc, // 5
        0x976EA74026E726554dB657fA54763abd0C3a0aa9, // 6
        0x14dC79964da2C08b23698B3D3cc7Ca32193d9955, // 7
        0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f, // 8
        0xa0Ee7A142d267C1f36714E4a8F75612F20a79720, // 9
        0xBcd4042DE499D14e55001CcbB24a551F3b954096, // 10
        0x71bE63f3384f5fb98995898A86B02Fb2426c5788, // 11
        0xFABB0ac9d68B0B445fB7357272Ff202C5651694a, // 12
        0x1CBd3b2770909D4e10f157cABC84C7264073C9Ec, // 13
        0xdF3e18d64BC6A983f673Ab319CCaE4f1a57C7097, // 14
        0xcd3B766CCDd6AE721141F452C550Ca635964ce71, // 15
        0x2546BcD3c84621e976D8185a91A922aE77ECEc30, // 16
        0xbDA5747bFD65F08deb54cb465eB87D40e51B197E, // 17
        0xdD2FD4581271e230360230F9337D5c0430Bf44C0, // 18
        0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199, // 19
        0x09DB0a93B389bEF724429898f539AEB7ac2Dd55f, // 20
        0x02484cb50AAC86Eae85610D6f4Bf026f30f6627D, // 21
        0x08135Da0A343E492FA2d4282F2AE34c6c5CC1BbE, // 22
        0x5E661B79FE2D3F6cE70F5AAC07d8Cd9abb2743F1, // 23
        0x61097BA76cD906d2ba4FD106E757f7Eb455fc295, // 24
        0xDf37F81dAAD2b0327A0A50003740e1C935C70913, // 25
        0x553BC17A05702530097c3677091C5BB47a3a7931, // 26
        0x87BdCE72c06C21cd96219BD8521bDF1F42C78b5e, // 27
        0x40Fc963A729c542424cD800349a7E4Ecc4896624, // 28
        0x9DCCe783B6464611f38631e6C851bf441907c710 // 29
    ];

    /// @notice The address of the deployer account.
    address internal deployer;

    /// @notice Sets up the script and ensures the deployer account is used to make calls.
    function setUp() public override {
        deployer = makeAddr("deployer");
        super.setUp();
    }

    function artifactDependencies() internal view returns (L1Dependencies memory l1Dependencies_) {
        return L1Dependencies({
            l1CrossDomainMessengerProxy: mustGetAddress("L1CrossDomainMessengerProxy"),
            l1StandardBridgeProxy: mustGetAddress("L1StandardBridgeProxy"),
            l1ERC721BridgeProxy: mustGetAddress("L1ERC721BridgeProxy")
        });
    }

    /// @notice The alloc object is sorted numerically by address.
    ///         Sets the precompiles, proxies, and the implementation accounts to be `vm.dumpState`
    ///         to generate a L2 genesis alloc.
    function runWithStateDump() public {
        runWithOptions(OutputMode.DEFAULT_LATEST, artifactDependencies());
    }

    /// @notice Alias for `runWithStateDump` so that no `--sig` needs to be specified.
    function run() public {
        runWithStateDump();
    }

    /// @notice This is used by op-e2e to have a version of the L2 allocs for each upgrade.
    function runWithAllUpgrades() public {
        runWithOptions(OutputMode.OUTPUT_ALL, artifactDependencies());
    }

    /// @notice Build the L2 genesis.
    function runWithOptions(OutputMode _mode, L1Dependencies memory _l1Dependencies) public {
        vm.startPrank(deployer);
        vm.chainId(cfg.l2ChainID());

        dealEthToPrecompiles();
        setPredeployProxies();
        setPredeployImplementations(_l1Dependencies);
        setPreinstalls();
        if (cfg.fundDevAccounts()) {
            fundDevAccounts();
        }
        vm.stopPrank();

        // Genesis is "complete" at this point, but some hardfork activation steps remain.
        // Depending on the "Output Mode" we perform the activations and output the necessary state dumps.
        if (_mode == OutputMode.LOCAL_DELTA) {
            return;
        }
        if (_mode == OutputMode.OUTPUT_ALL) {
            writeGenesisAllocs(Config.stateDumpPath("-delta"));
        }

        activateEcotone();

        if (_mode == OutputMode.LOCAL_ECOTONE) {
            return;
        }
        if (_mode == OutputMode.OUTPUT_ALL) {
            writeGenesisAllocs(Config.stateDumpPath("-ecotone"));
        }

        activateFjord();

        if (_mode == OutputMode.OUTPUT_ALL || _mode == OutputMode.DEFAULT_LATEST) {
            writeGenesisAllocs(Config.stateDumpPath(""));
        }
    }

    /// @notice Give all of the precompiles 1 wei
    function dealEthToPrecompiles() internal {
        console.log("Setting precompile 1 wei balances");
        for (uint256 i; i < PRECOMPILE_COUNT; i++) {
            vm.deal(address(uint160(i)), 1);
        }
    }

    /// @notice Set up the accounts that correspond to the predeploys.
    ///         The Proxy bytecode should be set. All proxied predeploys should have
    ///         the 1967 admin slot set to the ProxyAdmin predeploy. All defined predeploys
    ///         should have their implementations set.
    ///         Warning: the predeploy accounts have contract code, but 0 nonce value.
    function setPredeployProxies() public {
        console.log("Setting Predeploy proxies");
        bytes memory code = vm.getDeployedCode("Proxy.sol:Proxy");
        uint160 prefix = uint160(0x420) << 148;

        console.log(
            "Setting proxy deployed bytecode for addresses in range %s through %s",
            address(prefix | uint160(0)),
            address(prefix | uint160(Predeploys.PREDEPLOY_COUNT - 1))
        );
        for (uint256 i = 0; i < Predeploys.PREDEPLOY_COUNT; i++) {
            address addr = address(prefix | uint160(i));
            if (Predeploys.notProxied(addr)) {
                console.log("Skipping proxy at %s", addr);
                continue;
            }

            vm.etch(addr, code);
            EIP1967Helper.setAdmin(addr, Predeploys.PROXY_ADMIN);

            if (Predeploys.isSupportedPredeploy(addr, cfg.useInterop())) {
                address implementation = Predeploys.predeployToCodeNamespace(addr);
                console.log("Setting proxy %s implementation: %s", addr, implementation);
                EIP1967Helper.setImplementation(addr, implementation);
            }
        }
    }

    /// @notice Sets all the implementations for the predeploy proxies. For contracts without proxies,
    ///      sets the deployed bytecode at their expected predeploy address.
    ///      LEGACY_ERC20_ETH and L1_MESSAGE_SENDER are deprecated and are not set.
    function setPredeployImplementations(L1Dependencies memory _l1Dependencies) internal {
        console.log("Setting predeploy implementations with L1 contract dependencies:");
        console.log("- L1CrossDomainMessengerProxy: %s", _l1Dependencies.l1CrossDomainMessengerProxy);
        console.log("- L1StandardBridgeProxy: %s", _l1Dependencies.l1StandardBridgeProxy);
        console.log("- L1ERC721BridgeProxy: %s", _l1Dependencies.l1ERC721BridgeProxy);
        setLegacyMessagePasser(); // 0
        // 01: legacy, not used in OP-Stack
        setDeployerWhitelist(); // 2
        // 3,4,5: legacy, not used in OP-Stack.
        setWETH(); // 6: WETH (not behind a proxy)
        setL2CrossDomainMessenger(_l1Dependencies.l1CrossDomainMessengerProxy); // 7
        // 8,9,A,B,C,D,E: legacy, not used in OP-Stack.
        setGasPriceOracle(); // f
        setL2StandardBridge(_l1Dependencies.l1StandardBridgeProxy); // 10
        setSequencerFeeVault(); // 11
        setOptimismMintableERC20Factory(); // 12
        setL1BlockNumber(); // 13
        setL2ERC721Bridge(_l1Dependencies.l1ERC721BridgeProxy); // 14
        setL1Block(); // 15
        setL2ToL1MessagePasser(); // 16
        setOptimismMintableERC721Factory(); // 17
        setProxyAdmin(); // 18
        setBaseFeeVault(); // 19
        setL1FeeVault(); // 1A
        // 1B,1C,1D,1E,1F: not used.
        setSchemaRegistry(); // 20
        setEAS(); // 21
        setGovernanceToken(); // 42: OP (not behind a proxy)
        if (cfg.useInterop()) {
            setCrossL2Inbox(); // 22
            setL2ToL2CrossDomainMessenger(); // 23
        }
    }

    function setProxyAdmin() public {
        // Note the ProxyAdmin implementation itself is behind a proxy that owns itself.
        address impl = _setImplementationCode(Predeploys.PROXY_ADMIN);

        bytes32 _ownerSlot = bytes32(0);

        // there is no initialize() function, so we just set the storage manually.
        vm.store(Predeploys.PROXY_ADMIN, _ownerSlot, bytes32(uint256(uint160(cfg.proxyAdminOwner()))));
        // update the proxy to not be uninitialized (although not standard initialize pattern)
        vm.store(impl, _ownerSlot, bytes32(uint256(uint160(cfg.proxyAdminOwner()))));
    }

    function setL2ToL1MessagePasser() public {
        _setImplementationCode(Predeploys.L2_TO_L1_MESSAGE_PASSER);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL2CrossDomainMessenger(address payable _l1CrossDomainMessengerProxy) public {
        address impl = _setImplementationCode(Predeploys.L2_CROSS_DOMAIN_MESSENGER);

        L2CrossDomainMessenger(impl).initialize({ _l1CrossDomainMessenger: L1CrossDomainMessenger(address(0)) });

        L2CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).initialize({
            _l1CrossDomainMessenger: L1CrossDomainMessenger(_l1CrossDomainMessengerProxy)
        });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL2StandardBridge(address payable _l1StandardBridgeProxy) public {
        address impl = _setImplementationCode(Predeploys.L2_STANDARD_BRIDGE);

        L2StandardBridge(payable(impl)).initialize({ _otherBridge: L1StandardBridge(payable(address(0))) });

        L2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).initialize({
            _otherBridge: L1StandardBridge(_l1StandardBridgeProxy)
        });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL2ERC721Bridge(address payable _l1ERC721BridgeProxy) public {
        address impl = _setImplementationCode(Predeploys.L2_ERC721_BRIDGE);

        L2ERC721Bridge(impl).initialize({ _l1ERC721Bridge: payable(address(0)) });

        L2ERC721Bridge(Predeploys.L2_ERC721_BRIDGE).initialize({ _l1ERC721Bridge: payable(_l1ERC721BridgeProxy) });
    }

    /// @notice This predeploy is following the safety invariant #2,
    function setSequencerFeeVault() public {
        SequencerFeeVault vault = new SequencerFeeVault({
            _recipient: cfg.sequencerFeeVaultRecipient(),
            _minWithdrawalAmount: cfg.sequencerFeeVaultMinimumWithdrawalAmount(),
            _withdrawalNetwork: FeeVault.WithdrawalNetwork(cfg.sequencerFeeVaultWithdrawalNetwork())
        });

        address impl = Predeploys.predeployToCodeNamespace(Predeploys.SEQUENCER_FEE_WALLET);
        console.log("Setting %s implementation at: %s", "SequencerFeeVault", impl);
        vm.etch(impl, address(vault).code);

        /// Reset so its not included state dump
        vm.etch(address(vault), "");
        vm.resetNonce(address(vault));
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setOptimismMintableERC20Factory() public {
        address impl = _setImplementationCode(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY);

        OptimismMintableERC20Factory(impl).initialize({ _bridge: address(0) });

        OptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY).initialize({
            _bridge: Predeploys.L2_STANDARD_BRIDGE
        });
    }

    /// @notice This predeploy is following the safety invariant #2,
    function setOptimismMintableERC721Factory() public {
        OptimismMintableERC721Factory factory =
            new OptimismMintableERC721Factory({ _bridge: Predeploys.L2_ERC721_BRIDGE, _remoteChainId: cfg.l1ChainID() });

        address impl = Predeploys.predeployToCodeNamespace(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY);
        console.log("Setting %s implementation at: %s", "OptimismMintableERC721Factory", impl);
        vm.etch(impl, address(factory).code);

        /// Reset so its not included state dump
        vm.etch(address(factory), "");
        vm.resetNonce(address(factory));
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL1Block() public {
        _setImplementationCode(Predeploys.L1_BLOCK_ATTRIBUTES);
        // Note: L1 block attributes are set to 0.
        // Before the first user-tx the state is overwritten with actual L1 attributes.
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setGasPriceOracle() public {
        _setImplementationCode(Predeploys.GAS_PRICE_ORACLE);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setDeployerWhitelist() public {
        _setImplementationCode(Predeploys.DEPLOYER_WHITELIST);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract is NOT proxied and the state that is set
    ///         in the constructor is set manually.
    function setWETH() public {
        console.log("Setting %s implementation at: %s", "WETH", Predeploys.WETH);
        vm.etch(Predeploys.WETH, vm.getDeployedCode("WETH.sol:WETH"));
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL1BlockNumber() public {
        _setImplementationCode(Predeploys.L1_BLOCK_NUMBER);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setLegacyMessagePasser() public {
        _setImplementationCode(Predeploys.LEGACY_MESSAGE_PASSER);
    }

    /// @notice This predeploy is following the safety invariant #2.
    function setBaseFeeVault() public {
        BaseFeeVault vault = new BaseFeeVault({
            _recipient: cfg.baseFeeVaultRecipient(),
            _minWithdrawalAmount: cfg.baseFeeVaultMinimumWithdrawalAmount(),
            _withdrawalNetwork: FeeVault.WithdrawalNetwork(cfg.baseFeeVaultWithdrawalNetwork())
        });

        address impl = Predeploys.predeployToCodeNamespace(Predeploys.BASE_FEE_VAULT);
        console.log("Setting %s implementation at: %s", "BaseFeeVault", impl);
        vm.etch(impl, address(vault).code);

        /// Reset so its not included state dump
        vm.etch(address(vault), "");
        vm.resetNonce(address(vault));
    }

    /// @notice This predeploy is following the safety invariant #2.
    function setL1FeeVault() public {
        L1FeeVault vault = new L1FeeVault({
            _recipient: cfg.l1FeeVaultRecipient(),
            _minWithdrawalAmount: cfg.l1FeeVaultMinimumWithdrawalAmount(),
            _withdrawalNetwork: FeeVault.WithdrawalNetwork(cfg.l1FeeVaultWithdrawalNetwork())
        });

        address impl = Predeploys.predeployToCodeNamespace(Predeploys.L1_FEE_VAULT);
        console.log("Setting %s implementation at: %s", "L1FeeVault", impl);
        vm.etch(impl, address(vault).code);

        /// Reset so its not included state dump
        vm.etch(address(vault), "");
        vm.resetNonce(address(vault));
    }

    /// @notice This predeploy is following the safety invariant #2.
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

    /// @notice This predeploy is following the safety invariant #1.
    function setSchemaRegistry() public {
        _setImplementationCode(Predeploys.SCHEMA_REGISTRY);
    }

    /// @notice This predeploy is following the safety invariant #2,
    ///         It uses low level create to deploy the contract due to the code
    ///         having immutables and being a different compiler version.
    function setEAS() public {
        string memory cname = Predeploys.getName(Predeploys.EAS);
        address impl = Predeploys.predeployToCodeNamespace(Predeploys.EAS);
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

    /// @notice This predeploy is following the saftey invariant #2.
    ///         This contract has no initializer.
    function setCrossL2Inbox() internal {
        _setImplementationCode(Predeploys.CROSS_L2_INBOX);
    }

    /// @notice This predeploy is following the saftey invariant #2.
    ///         This contract has no initializer.
    function setL2ToL2CrossDomainMessenger() internal {
        _setImplementationCode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
    }

    /// @notice Sets all the preinstalls.
    ///         Warning: the creator-accounts of the preinstall contracts have 0 nonce values.
    ///         When performing a regular user-initiated contract-creation of a preinstall,
    ///         the creation will fail (but nonce will be bumped and not blocked).
    ///         The preinstalls themselves are all inserted with a nonce of 1, reflecting regular user execution.
    function setPreinstalls() internal {
        _setPreinstallCode(Preinstalls.MultiCall3);
        _setPreinstallCode(Preinstalls.Create2Deployer);
        _setPreinstallCode(Preinstalls.Safe_v130);
        _setPreinstallCode(Preinstalls.SafeL2_v130);
        _setPreinstallCode(Preinstalls.MultiSendCallOnly_v130);
        _setPreinstallCode(Preinstalls.SafeSingletonFactory);
        _setPreinstallCode(Preinstalls.DeterministicDeploymentProxy);
        _setPreinstallCode(Preinstalls.MultiSend_v130);
        _setPreinstallCode(Preinstalls.Permit2);
        _setPreinstallCode(Preinstalls.SenderCreator);
        _setPreinstallCode(Preinstalls.EntryPoint); // ERC 4337
        _setPreinstallCode(Preinstalls.BeaconBlockRoots);
        // 4788 sender nonce must be incremented, since it's part of later upgrade-transactions.
        // For the upgrade-tx to not create a contract that conflicts with an already-existing copy,
        // the nonce must be bumped.
        vm.setNonce(Preinstalls.BeaconBlockRootsSender, 1);
    }

    /// @notice Activate Ecotone network upgrade.
    function activateEcotone() public {
        require(Preinstalls.BeaconBlockRoots.code.length > 0, "L2Genesis: must have beacon-block-roots contract");
        console.log("Activating ecotone in GasPriceOracle contract");

        vm.prank(L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).DEPOSITOR_ACCOUNT());
        GasPriceOracle(Predeploys.GAS_PRICE_ORACLE).setEcotone();
    }

    function activateFjord() public {
        console.log("Activating fjord in GasPriceOracle contract");
        vm.prank(L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).DEPOSITOR_ACCOUNT());
        GasPriceOracle(Predeploys.GAS_PRICE_ORACLE).setFjord();
    }

    /// @notice Sets the bytecode in state
    function _setImplementationCode(address _addr) internal returns (address) {
        string memory cname = Predeploys.getName(_addr);
        address impl = Predeploys.predeployToCodeNamespace(_addr);
        console.log("Setting %s implementation at: %s", cname, impl);
        vm.etch(impl, vm.getDeployedCode(string.concat(cname, ".sol:", cname)));
        return impl;
    }

    /// @notice Sets the bytecode in state
    function _setPreinstallCode(address _addr) internal {
        string memory cname = Preinstalls.getName(_addr);
        console.log("Setting %s preinstall code at: %s", cname, _addr);
        vm.etch(_addr, Preinstalls.getDeployedCode(_addr, cfg.l2ChainID()));
        // during testing in a shared L1/L2 account namespace some preinstalls may already have been inserted and used.
        if (vm.getNonce(_addr) == 0) {
            vm.setNonce(_addr, 1);
        }
    }

    /// @notice Writes the genesis allocs, i.e. the state dump, to disk
    function writeGenesisAllocs(string memory _path) public {
        /// Reset so its not included state dump
        vm.etch(address(cfg), "");
        vm.etch(msg.sender, "");
        vm.resetNonce(msg.sender);
        vm.deal(msg.sender, 0);

        vm.deal(deployer, 0);
        vm.resetNonce(deployer);

        console.log("Writing state dump to: %s", _path);
        vm.dumpState(_path);
        sortJsonByKeys(_path);
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

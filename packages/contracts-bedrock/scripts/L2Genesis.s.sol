// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { Script } from "forge-std/Script.sol";
import { SetPreinstalls } from "scripts/SetPreinstalls.s.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { OutputMode, OutputModeUtils, Fork, ForkUtils } from "scripts/libraries/Config.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { Types } from "src/libraries/Types.sol";
import { L2ContractsManagerTypes } from "src/libraries/L2ContractsManagerTypes.sol";
import { L2ContractsManager } from "src/L2/L2ContractsManager.sol";

// Interfaces
import { IOptimismMintableERC721Factory } from "interfaces/L2/IOptimismMintableERC721Factory.sol";
import { IGovernanceToken } from "interfaces/governance/IGovernanceToken.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IL2StandardBridge } from "interfaces/L2/IL2StandardBridge.sol";
import { IL2ERC721Bridge } from "interfaces/L2/IL2ERC721Bridge.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IERC721Bridge } from "interfaces/universal/IERC721Bridge.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL2CrossDomainMessenger } from "interfaces/L2/IL2CrossDomainMessenger.sol";
import { IGasPriceOracle } from "interfaces/L2/IGasPriceOracle.sol";
import { IL1Block } from "interfaces/L2/IL1Block.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { IL1BlockCGT } from "interfaces/L2/IL1BlockCGT.sol";
import { IL2DevFeatureFlags } from "interfaces/L2/IL2DevFeatureFlags.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { IEventReplayer } from "interfaces/private-interop/IEventReplayer.sol";
import { INativeMintBridge } from "interfaces/private-interop/INativeMintBridge.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Features } from "src/libraries/Features.sol";

/// @title L2Genesis
/// @notice Generates the genesis state for the L2 network.
///         The following safety invariants are used when setting state:
///         1. `vm.getDeployedBytecode` can only be used with `vm.etch` when there are no side
///         effects in the constructor and no immutables in the bytecode.
///         2. A contract must be deployed using the `new` syntax if there are immutables in the code.
///         Any other side effects from the init code besides setting the immutables must be cleaned up afterwards.
contract L2Genesis is Script {
    struct Input {
        uint256 l1ChainID;
        uint256 l2ChainID;
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
        address operatorFeeVaultRecipient;
        uint256 operatorFeeVaultMinimumWithdrawalAmount;
        uint256 operatorFeeVaultWithdrawalNetwork;
        address governanceTokenOwner;
        uint256 fork;
        bool enableGovernance;
        bool fundDevAccounts;
        bool useCustomGasToken;
        bool useInterop;
        string gasPayingTokenName;
        string gasPayingTokenSymbol;
        uint256 nativeAssetLiquidityAmount;
        address liquidityControllerOwner;
        bytes32 devFeatureBitmap;
        uint256 privateInteropCounterpartyChainID;
        address privateInteropLockVault;
    }

    using ForkUtils for Fork;
    using OutputModeUtils for OutputMode;

    uint256 internal constant PRECOMPILE_COUNT = 256;

    uint80 internal constant DEV_ACCOUNT_FUND_AMT = 10_000 ether;
    uint32 internal constant WITHDRAWAL_MIN_GAS_LIMIT = 800_000;
    uint256 internal constant MIN_WITHDRAWAL_AMOUNT_THRESHOLD = 2 ether;

    /// @notice CREATE2 salt for the throwaway L2ContractsManager.
    bytes32 internal constant L2CM_SALT = bytes32(uint256(keccak256("optimism.l2genesis.l2cm")));

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

    /// @notice Alias for `runWithStateDump` so that no `--sig` needs to be specified.
    function run(Input memory _input) public {
        require(
            _input.useInterop
                == DevFeatures.isDevFeatureEnabled(_input.devFeatureBitmap, DevFeatures.OPTIMISM_PORTAL_INTEROP),
            "L2Genesis: useInterop and OPTIMISM_PORTAL_INTEROP devFeature bit must agree"
        );
        _checkPrivateInteropInput(_input);
        address deployer = makeAddr("deployer");
        vm.startPrank(deployer);
        vm.chainId(_input.l2ChainID);

        dealEthToPrecompiles();
        setPredeployProxies(_input);
        vm.stopPrank();

        // Set L1 Block has its own pranking requirements which it handles internally
        setPredeployImplementations(_input);

        vm.startPrank(deployer);
        setPreinstalls();
        if (_input.fundDevAccounts) {
            fundDevAccounts();
        }

        vm.stopPrank();
        vm.deal(deployer, 0);
        vm.resetNonce(deployer);

        Fork _fork = Fork(_input.fork);

        if (forkEquals(_fork, Fork.DELTA)) {
            return;
        }

        activateEcotone();

        if (forkEquals(_fork, Fork.ECOTONE)) {
            return;
        }

        activateFjord();

        if (forkEquals(_fork, Fork.FJORD)) {
            return;
        }

        if (forkEquals(_fork, Fork.GRANITE)) {
            return;
        }

        if (forkEquals(_fork, Fork.HOLOCENE)) {
            return;
        }

        activateIsthmus();

        if (forkEquals(_fork, Fork.ISTHMUS)) {
            return;
        }

        activateJovian();

        if (forkEquals(_fork, Fork.JOVIAN)) {
            return;
        }

        if (forkEquals(_fork, Fork.KARST)) {
            return;
        }

        if (forkEquals(_fork, Fork.LAGOON)) {
            return;
        }
    }

    function forkEquals(Fork _latest, Fork _current) internal pure returns (bool) {
        return _latest == _current;
    }

    /// @notice Give all of the precompiles 1 wei
    function dealEthToPrecompiles() internal {
        for (uint256 i; i < PRECOMPILE_COUNT; i++) {
            vm.deal(address(uint160(i)), 1);
        }
    }

    /// @notice Set up the accounts that correspond to the predeploys.
    ///         The Proxy bytecode should be set. All proxied predeploys should have
    ///         the 1967 admin slot set to the L2ProxyAdmin predeploy. All defined predeploys
    ///         should have their implementations set.
    ///         Warning: the predeploy accounts have contract code, but 0 nonce value, contrary
    ///         to the expected nonce of 1 per EIP-161. This is because the legacy go genesis
    //          script didn't set the nonce and we didn't want to change that behavior when
    ///         migrating genesis generation to Solidity.
    function setPredeployProxies(Input memory _input) internal {
        bytes memory code = DeployUtils.getDeployedCode("Proxy");
        uint160 prefix = uint160(0x420) << 148;

        for (uint256 i = 0; i < Predeploys.PREDEPLOY_COUNT; i++) {
            address addr = address(prefix | uint160(i));

            // Non-proxied predeploys are excluded from proxy setup.
            if (Predeploys.notProxied(addr)) continue;

            vm.etch(addr, code);
            EIP1967Helper.setAdmin(addr, Predeploys.PROXY_ADMIN);

            if (
                Predeploys.isSupportedPredeploy(
                    addr, _input.fork, _input.useCustomGasToken, _input.useInterop, _input.devFeatureBitmap
                )
            ) {
                address implementation = Predeploys.predeployToCodeNamespace(addr);
                EIP1967Helper.setImplementation(addr, implementation);
            }
        }
    }

    /// @notice Sets all the implementations for the predeploy proxies. For contracts without proxies,
    ///      sets the deployed bytecode at their expected predeploy address.
    ///      LEGACY_ERC20_ETH and L1_MESSAGE_SENDER are deprecated and are not set.
    function setPredeployImplementations(Input memory _input) internal {
        // Contracts with initializers now call
        // _assertOnlyProxyAdminOrProxyAdminOwner(). Prank as the proxy admin owner so that those
        // assertions pass for both proxy and implementation initialization calls.
        vm.startPrank(_input.opChainProxyAdminOwner);
        // Must be first: other contracts' initialize() calls assert _assertOnlyProxyAdminOrProxyAdminOwner(),
        // which reads L2ProxyAdmin.owner(). The owner slot must be set before any initializer runs.
        setL2ProxyAdmin(_input); // 18
        setLegacyMessagePasser(); // 0: LEGACY_MESSAGE_PASSER is deprecated and not used in OP-Stack
        // 01: legacy, not used in OP-Stack
        setDeployerWhitelist(); // 2: DEPLOYER_WHITELIST is deprecated and not used in OP-Stack
        // 3,4,5: legacy, not used in OP-Stack.
        setWETH(); // 6: WETH (not behind a proxy)
        setL2CrossDomainMessenger(); // 7
        // 8,9,A,B,C,D,E: legacy, not used in OP-Stack.
        setGasPriceOracle(); // f
        setL2StandardBridge(); // 10
        setSequencerFeeVault(_input); // 11
        setOptimismMintableERC20Factory(); // 12
        setL1BlockNumber(); // 13: L1_BLOCK_NUMBER is deprecated and not used in OP-Stack
        setL2ERC721Bridge(); // 14
        setL1Block(_input); // 15
        setL2ToL1MessagePasser(_input.useCustomGasToken); // 16
        setOptimismMintableERC721Factory(); // 17
        setBaseFeeVault(_input); // 19
        setL1FeeVault(_input); // 1A
        setOperatorFeeVault(_input); // 1B
        // 1C,1D,1E,1F: not used.
        setSchemaRegistry(); // 20
        setEAS(); // 21
        setGovernanceToken(_input); // 42: OP (not behind a proxy)
        if (_isGenesisInteropEnabled(_input)) {
            // Both flags must be explicitly set in order to enable Interop
            setCrossL2Inbox(); // 22
            setL2ToL2CrossDomainMessenger(); // 23
            setSuperchainETHBridge(); // 24
            setETHLiquidity(); // 25
        }
        if (_input.useCustomGasToken) {
            setLiquidityController(_input); // 29
            setNativeAssetLiquidity(_input); // 2A
        }
        vm.stopPrank();
        // The pranked `create` calls in setEAS() and setGovernanceToken() bump the proxy admin
        // owner's nonce. Reset it so the account does not appear in the genesis state dump.
        vm.resetNonce(_input.opChainProxyAdminOwner);
        // These calls don't need the opChainProxyAdminOwner prank: setConditionalDeployer uses
        // vm.etch and setL2DevFeatureFlags manages its own prank as DEPOSITOR_ACCOUNT.
        setConditionalDeployer(); // 2C
        setL2DevFeatureFlags(_input); // 2D
        // deploy() upgrades ConditionalDeployer and L2DevFeatureFlags, so it must run after both are set.
        _deployPredeploysViaL2CM(_input);
        // Private interop runs last, after a complete stock genesis exists. The rendering half
        // REPLACES an implementation the L2ContractsManager has just installed, and the private
        // half authorizes a minter on a LiquidityController that L2CM has just initialized, so
        // neither can observe a half-finished state.
        if (_isPrivateInteropRendering(_input)) {
            setPrivateInteropRendering();
        }
        if (_isPrivateInteropPrivateChain(_input)) {
            setPrivateInteropPrivateChain(_input);
        }
    }

    /// @notice Validates the private interop half selection before any state is written.
    /// @dev The two halves share a chain ID but are different chains in content, and a genesis that
    ///      claimed to be both would be neither. The rendering carries no custom gas token: its
    ///      replay transactions are zero-priced and the rendering starts with a zero base fee.
    function _checkPrivateInteropInput(Input memory _input) internal pure {
        bool rendering = _isPrivateInteropRendering(_input);
        bool privateChain = _isPrivateInteropPrivateChain(_input);

        require(
            !(rendering && privateChain),
            "L2Genesis: PRIVATE_INTEROP_RENDERING and PRIVATE_INTEROP_PRIVATE_CHAIN are mutually exclusive"
        );

        if (rendering) {
            require(
                _isGenesisInteropEnabled(_input), "L2Genesis: private interop rendering requires interop at genesis"
            );
            require(
                !_input.useCustomGasToken, "L2Genesis: private interop rendering must not be a custom gas token chain"
            );
        }

        if (privateChain) {
            require(
                _isGenesisInteropEnabled(_input), "L2Genesis: private interop private chain requires interop at genesis"
            );
            require(_input.useCustomGasToken, "L2Genesis: private interop private chain requires a custom gas token");
            require(
                _input.privateInteropCounterpartyChainID != 0,
                "L2Genesis: private interop counterparty chain ID must be set"
            );
            require(_input.privateInteropLockVault != address(0), "L2Genesis: private interop lock vault must be set");
        }
    }

    /// @notice Returns true when this genesis is the PUBLIC RENDERING half of a private interop pair.
    function _isPrivateInteropRendering(Input memory _input) internal pure returns (bool) {
        return DevFeatures.isDevFeatureEnabled(_input.devFeatureBitmap, DevFeatures.PRIVATE_INTEROP_RENDERING);
    }

    /// @notice Returns true when this genesis is the PRIVATE half of a private interop pair.
    function _isPrivateInteropPrivateChain(Input memory _input) internal pure returns (bool) {
        return DevFeatures.isDevFeatureEnabled(_input.devFeatureBitmap, DevFeatures.PRIVATE_INTEROP_PRIVATE_CHAIN);
    }

    /// @notice Builds the implementation records for the temporary L2ContractsManager.
    /// @dev StorageSetter has no predeploy address, so it is passed as a zero record that deploy mode
    ///      ignores. Every variant is included because the L2CM constructor resolves each record by name;
    ///      both variants of a proxy map to the same code-namespace address.
    function _buildL2CMImplRecords() internal pure returns (L2ContractsManagerTypes.ImplRecord[] memory records_) {
        Predeploys.PredeployRecord[] memory upgradeable = Predeploys.getUpgradeableRecords();

        uint256 count = 1; // StorageSetter
        for (uint256 i = 0; i < upgradeable.length; i++) {
            count += upgradeable[i].variants.length;
        }

        records_ = new L2ContractsManagerTypes.ImplRecord[](count);
        records_[0] = L2ContractsManagerTypes.ImplRecord({ name: "StorageSetter", impl: address(0) });
        uint256 idx = 1;
        for (uint256 i = 0; i < upgradeable.length; i++) {
            address impl = Predeploys.predeployToCodeNamespace(upgradeable[i].proxy);
            for (uint256 j = 0; j < upgradeable[i].variants.length; j++) {
                records_[idx++] =
                    L2ContractsManagerTypes.ImplRecord({ name: upgradeable[i].variants[j].name, impl: impl });
            }
        }
    }

    /// @notice Builds the L2ContractsManager config from the genesis input.
    function _buildL2CMConfig(Input memory _input)
        internal
        pure
        returns (L2ContractsManagerTypes.FullConfig memory config_)
    {
        config_.crossDomainMessenger = L2ContractsManagerTypes.CrossDomainMessengerConfig({
            otherMessenger: ICrossDomainMessenger(_input.l1CrossDomainMessengerProxy)
        });
        config_.standardBridge =
            L2ContractsManagerTypes.StandardBridgeConfig({ otherBridge: IStandardBridge(_input.l1StandardBridgeProxy) });
        config_.erc721Bridge =
            L2ContractsManagerTypes.ERC721BridgeConfig({ otherBridge: IERC721Bridge(_input.l1ERC721BridgeProxy) });
        config_.mintableERC20Factory =
            L2ContractsManagerTypes.MintableERC20FactoryConfig({ bridge: Predeploys.L2_STANDARD_BRIDGE });
        config_.mintableERC721Factory = L2ContractsManagerTypes.MintableERC721FactoryConfig({
            bridge: Predeploys.L2_ERC721_BRIDGE,
            remoteChainID: _input.l1ChainID
        });
        config_.sequencerFeeVault = L2ContractsManagerTypes.FeeVaultConfig({
            recipient: _input.sequencerFeeVaultRecipient,
            minWithdrawalAmount: _input.sequencerFeeVaultMinimumWithdrawalAmount,
            withdrawalNetwork: Types.WithdrawalNetwork(_input.sequencerFeeVaultWithdrawalNetwork)
        });
        config_.baseFeeVault = L2ContractsManagerTypes.FeeVaultConfig({
            recipient: _input.baseFeeVaultRecipient,
            minWithdrawalAmount: _input.baseFeeVaultMinimumWithdrawalAmount,
            withdrawalNetwork: Types.WithdrawalNetwork(_input.baseFeeVaultWithdrawalNetwork)
        });
        config_.l1FeeVault = L2ContractsManagerTypes.FeeVaultConfig({
            recipient: _input.l1FeeVaultRecipient,
            minWithdrawalAmount: _input.l1FeeVaultMinimumWithdrawalAmount,
            withdrawalNetwork: Types.WithdrawalNetwork(_input.l1FeeVaultWithdrawalNetwork)
        });
        config_.operatorFeeVault = L2ContractsManagerTypes.FeeVaultConfig({
            recipient: _input.operatorFeeVaultRecipient,
            minWithdrawalAmount: _input.operatorFeeVaultMinimumWithdrawalAmount,
            withdrawalNetwork: Types.WithdrawalNetwork(_input.operatorFeeVaultWithdrawalNetwork)
        });
        config_.liquidityController = L2ContractsManagerTypes.LiquidityControllerConfig({
            owner: _input.liquidityControllerOwner,
            gasPayingTokenName: _input.gasPayingTokenName,
            gasPayingTokenSymbol: _input.gasPayingTokenSymbol
        });
        config_.isCustomGasToken = _input.useCustomGasToken;
        config_.isInterop = _isGenesisInteropEnabled(_input);
    }

    /// @notice Deterministic CREATE2 address of the throwaway L2ContractsManager.
    /// @dev Pre-computable from the salt and init code, so consumers don't depend on deploy ordering
    ///      (e.g. nonce). Shares a compilation unit with the `new ... { salt }` deploy, so the init code matches.
    function temporaryL2CMAddress() public view returns (address) {
        bytes32 initCodeHash =
            keccak256(abi.encodePacked(type(L2ContractsManager).creationCode, abi.encode(_buildL2CMImplRecords())));
        // Plain CREATE2 derivation (keccak256(0xff ++ deployer ++ salt ++ initCodeHash)). Avoids the
        // vm.computeCreate2Address cheatcode, which the op-deployer script host does not implement.
        return
            address(uint160(uint256(keccak256(abi.encodePacked(bytes1(0xff), address(this), L2CM_SALT, initCodeHash)))));
    }

    /// @notice Upgrades, initializes, and configures predeploy proxies via the L2ContractsManager.
    /// @dev Deploys a throwaway L2CM, points the L2ProxyAdmin proxy's implementation at it, and calls
    ///      deploy() through the proxy.
    function _deployPredeploysViaL2CM(Input memory _input) internal {
        L2ContractsManager l2cm = new L2ContractsManager{ salt: L2CM_SALT }(_buildL2CMImplRecords());
        L2ContractsManagerTypes.FullConfig memory config = _buildL2CMConfig(_input);

        EIP1967Helper.setImplementation(Predeploys.PROXY_ADMIN, address(l2cm));
        L2ContractsManager(Predeploys.PROXY_ADMIN).deploy(config);

        // Assert deploy() restored the ProxyAdmin impl in place of the the temp L2CM.
        require(
            EIP1967Helper.getImplementation(Predeploys.PROXY_ADMIN)
                == Predeploys.predeployToCodeNamespace(Predeploys.PROXY_ADMIN),
            "L2Genesis: L2ProxyAdmin implementation not restored after deploy()"
        );

        vm.etch(address(l2cm), "");
        vm.resetNonce(address(l2cm));
    }

    /// @notice Returns true when interop should be active in the genesis state.
    function _isGenesisInteropEnabled(Input memory _input) internal pure returns (bool) {
        return _input.fork >= uint256(Fork.LAGOON) && _input.useInterop
            && DevFeatures.isDevFeatureEnabled(_input.devFeatureBitmap, DevFeatures.OPTIMISM_PORTAL_INTEROP);
    }

    function setInteropPredeployProxies() internal { }

    /// @notice This predeploy is following the safety invariant #2.
    ///         Follows invariant #2 since the constructor transfers ownership to the input owner,
    ///         and therefore requires setting the storage manually here.
    function setL2ProxyAdmin(Input memory _input) internal {
        // Note the L2ProxyAdmin implementation itself is behind a proxy that owns itself.
        _setImplementationCode(Predeploys.PROXY_ADMIN);

        bytes32 _ownerSlot = bytes32(0);

        // L2ProxyAdmin has no initializer by design, so we set the proxy owner slot directly.
        vm.store(Predeploys.PROXY_ADMIN, _ownerSlot, bytes32(uint256(uint160(_input.opChainProxyAdminOwner))));
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL2ToL1MessagePasser(bool _useCustomGasToken) internal {
        if (_useCustomGasToken) {
            string memory cname = "L2ToL1MessagePasserCGT";
            address impl = Predeploys.predeployToCodeNamespace(Predeploys.L2_TO_L1_MESSAGE_PASSER);
            vm.etch(impl, DeployUtils.getDeployedCode(cname));
        } else {
            _setImplementationCode(Predeploys.L2_TO_L1_MESSAGE_PASSER);
        }
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL2CrossDomainMessenger() internal {
        address impl = _setImplementationCode(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        IL2CrossDomainMessenger(impl).initialize({ _l1CrossDomainMessenger: ICrossDomainMessenger(address(0)) });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL2StandardBridge() internal {
        address impl = _setImplementationCode(Predeploys.L2_STANDARD_BRIDGE);
        IL2StandardBridge(payable(impl)).initialize({ _otherBridge: IStandardBridge(payable(address(0))) });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL2ERC721Bridge() internal {
        address impl = _setImplementationCode(Predeploys.L2_ERC721_BRIDGE);
        IL2ERC721Bridge(impl).initialize({ _l1ERC721Bridge: payable(address(0)) });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setSequencerFeeVault(Input memory _input) internal {
        _setFeeVault({
            _vaultAddr: Predeploys.SEQUENCER_FEE_WALLET,
            _useCustomGasToken: _input.useCustomGasToken,
            _withdrawalNetwork: Types.WithdrawalNetwork(_input.sequencerFeeVaultWithdrawalNetwork)
        });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setOptimismMintableERC20Factory() internal {
        address impl = _setImplementationCode(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY);
        IOptimismMintableERC20Factory(impl).initialize({ _bridge: address(0) });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setOptimismMintableERC721Factory() internal {
        address impl = _setImplementationCode(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY);
        IOptimismMintableERC721Factory(impl).initialize({ _bridge: address(0), _remoteChainID: 0 });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL1Block(Input memory _input) internal {
        if (_input.useCustomGasToken) {
            // Set the implementation code for L1BlockCGT
            string memory cname = "L1BlockCGT";
            address impl = Predeploys.predeployToCodeNamespace(Predeploys.L1_BLOCK_ATTRIBUTES);
            vm.etch(impl, DeployUtils.getDeployedCode(cname));

            // Set the custom gas token flag
            IL1BlockCGT(Predeploys.L1_BLOCK_ATTRIBUTES).setFeature(Features.CUSTOM_GAS_TOKEN);
        } else {
            _setImplementationCode(Predeploys.L1_BLOCK_ATTRIBUTES);
        }
        // Only set the runtime INTEROP feature flag at genesis if the chain is being born at or
        // beyond the Lagoon fork.
        if (_isGenesisInteropEnabled(_input)) {
            IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).setFeature(Features.INTEROP);
        }
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setGasPriceOracle() internal {
        _setImplementationCode(Predeploys.GAS_PRICE_ORACLE);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setDeployerWhitelist() internal {
        _setImplementationCode(Predeploys.DEPLOYER_WHITELIST);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract is NOT proxied and the state that is set
    ///         in the constructor is set manually.
    function setWETH() internal {
        vm.etch(Predeploys.WETH, DeployUtils.getDeployedCode("WETH"));
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL1BlockNumber() internal {
        _setImplementationCode(Predeploys.L1_BLOCK_NUMBER);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setLegacyMessagePasser() internal {
        _setImplementationCode(Predeploys.LEGACY_MESSAGE_PASSER);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setBaseFeeVault(Input memory _input) internal {
        _setFeeVault({
            _vaultAddr: Predeploys.BASE_FEE_VAULT,
            _useCustomGasToken: _input.useCustomGasToken,
            _withdrawalNetwork: Types.WithdrawalNetwork(_input.baseFeeVaultWithdrawalNetwork)
        });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL1FeeVault(Input memory _input) internal {
        _setFeeVault({
            _vaultAddr: Predeploys.L1_FEE_VAULT,
            _useCustomGasToken: _input.useCustomGasToken,
            _withdrawalNetwork: Types.WithdrawalNetwork(_input.l1FeeVaultWithdrawalNetwork)
        });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setOperatorFeeVault(Input memory _input) internal {
        _setFeeVault({
            _vaultAddr: Predeploys.OPERATOR_FEE_VAULT,
            _useCustomGasToken: _input.useCustomGasToken,
            _withdrawalNetwork: Types.WithdrawalNetwork(_input.operatorFeeVaultWithdrawalNetwork)
        });
    }

    /// @notice This predeploy is following the safety invariant #2.
    function setGovernanceToken(Input memory _input) internal {
        if (!_input.enableGovernance) {
            return;
        }

        IGovernanceToken token = IGovernanceToken(
            DeployUtils.create1({
                _name: "GovernanceToken",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IGovernanceToken.__constructor__, ()))
            })
        );
        vm.etch(Predeploys.GOVERNANCE_TOKEN, address(token).code);

        bytes32 _nameSlot = hex"0000000000000000000000000000000000000000000000000000000000000003";
        bytes32 _symbolSlot = hex"0000000000000000000000000000000000000000000000000000000000000004";
        bytes32 _ownerSlot = hex"000000000000000000000000000000000000000000000000000000000000000a";

        vm.store(Predeploys.GOVERNANCE_TOKEN, _nameSlot, vm.load(address(token), _nameSlot));
        vm.store(Predeploys.GOVERNANCE_TOKEN, _symbolSlot, vm.load(address(token), _symbolSlot));
        vm.store(Predeploys.GOVERNANCE_TOKEN, _ownerSlot, bytes32(uint256(uint160(_input.governanceTokenOwner))));

        /// Reset so its not included state dump
        vm.etch(address(token), "");
        vm.resetNonce(address(token));
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setSchemaRegistry() internal {
        _setImplementationCode(Predeploys.SCHEMA_REGISTRY);
    }

    /// @notice This predeploy is following the safety invariant #2,
    ///         It uses low level create to deploy the contract due to the code
    ///         having immutables and being a different compiler version.
    function setEAS() internal {
        string memory cname = Predeploys.getName(Predeploys.EAS);
        address impl = Predeploys.predeployToCodeNamespace(Predeploys.EAS);
        bytes memory code = DeployUtils.getCode(cname);

        address eas;
        assembly {
            eas := create(0, add(code, 0x20), mload(code))
        }

        vm.etch(impl, eas.code);

        /// Reset so its not included state dump
        vm.etch(address(eas), "");
        vm.resetNonce(address(eas));
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setCrossL2Inbox() internal {
        Predeploys.assertGates(Predeploys.CROSS_L2_INBOX, DevFeatures.OPTIMISM_PORTAL_INTEROP, false, true);
        _setImplementationCode(Predeploys.CROSS_L2_INBOX);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setL2ToL2CrossDomainMessenger() internal {
        Predeploys.assertGates(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, DevFeatures.OPTIMISM_PORTAL_INTEROP, false, true
        );
        _setImplementationCode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setETHLiquidity() internal {
        Predeploys.assertGates(Predeploys.ETH_LIQUIDITY, DevFeatures.OPTIMISM_PORTAL_INTEROP, false, true);
        _setImplementationCode(Predeploys.ETH_LIQUIDITY);
        vm.deal(Predeploys.ETH_LIQUIDITY, type(uint128).max);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setSuperchainETHBridge() internal {
        Predeploys.assertGates(Predeploys.SUPERCHAIN_ETH_BRIDGE, DevFeatures.OPTIMISM_PORTAL_INTEROP, false, true);
        _setImplementationCode(Predeploys.SUPERCHAIN_ETH_BRIDGE);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setLiquidityController(Input memory _input) internal {
        Predeploys.assertGates(Predeploys.LIQUIDITY_CONTROLLER, bytes32(0), true, false);
        address impl = _setImplementationCode(Predeploys.LIQUIDITY_CONTROLLER);

        ILiquidityController(impl).initialize({
            _owner: _input.liquidityControllerOwner,
            _gasPayingTokenName: "",
            _gasPayingTokenSymbol: ""
        });
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setNativeAssetLiquidity(Input memory _input) internal {
        Predeploys.assertGates(Predeploys.NATIVE_ASSET_LIQUIDITY, bytes32(0), true, false);
        _setImplementationCode(Predeploys.NATIVE_ASSET_LIQUIDITY);

        require(
            _input.nativeAssetLiquidityAmount <= type(uint248).max,
            "L2Genesis: native asset liquidity amount must be less than or equal to type(uint248).max"
        );

        // Pre-fund the liquidity contract with the specified amount
        vm.deal(Predeploys.NATIVE_ASSET_LIQUIDITY, _input.nativeAssetLiquidityAmount);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setConditionalDeployer() internal {
        Predeploys.assertGates(Predeploys.CONDITIONAL_DEPLOYER, bytes32(0), false, false);
        _setImplementationCode(Predeploys.CONDITIONAL_DEPLOYER);
    }

    /// @notice Sets up the L2DevFeatureFlags predeploy with the development feature bitmap.
    function setL2DevFeatureFlags(Input memory _input) internal {
        Predeploys.assertGates(Predeploys.L2_DEV_FEATURE_FLAGS, bytes32(0), false, false);
        _setImplementationCode(Predeploys.L2_DEV_FEATURE_FLAGS);
        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        IL2DevFeatureFlags(Predeploys.L2_DEV_FEATURE_FLAGS).setDevFeatureBitmap(_input.devFeatureBitmap);
    }

    /// @notice Renders the PUBLIC RENDERING half of a private interop pair.
    /// @dev The rendering is a derived-only chain whose blocks are a deterministic function of a
    ///      private chain's messenger traffic. It is a stock interop chain except in three places:
    ///      the messenger predeploy carries the replay implementation, the ClaimRegistry holds the
    ///      batcher's per-range commitments, and the EventReplayer re-emits everything the export
    ///      policy makes public. Authorization is inherited from the outer L1 batch transaction;
    ///      deposits are disabled, so there is no second transaction ingress or operator role.
    function setPrivateInteropRendering() internal {
        installReplayMessenger();
        setClaimRegistry();
        setEventReplayer();
    }

    /// @notice Installs the replay implementation at the standard L2ToL2CrossDomainMessenger
    ///         predeploy address, replacing the stock implementation the L2ContractsManager put
    ///         there moments ago.
    /// @dev Installing at the STANDARD address is the whole point: a replayed `SentMessage` then
    ///      carries the emitter every stock consumer already expects, so the message database, the
    ///      cross-safety judge and counterparty relayers need to know nothing about renderings.
    function installReplayMessenger() internal {
        address impl = Predeploys.predeployToCodeNamespace(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vm.etch(impl, DeployUtils.getDeployedCode("L2ToL2CrossDomainMessengerReplay"));
        EIP1967Helper.setAdmin(impl, Predeploys.PROXY_ADMIN);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setClaimRegistry() internal {
        _setPrivateInteropImplementation(Predeploys.CLAIM_REGISTRY, DeployUtils.getDeployedCode("ClaimRegistry"));
    }

    /// @notice This predeploy is following the safety invariant #2.
    function setEventReplayer() internal {
        IEventReplayer replayer = IEventReplayer(
            DeployUtils.create1({
                _name: "EventReplayer",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IEventReplayer.__constructor__, ()))
            })
        );
        _setPrivateInteropImplementation(Predeploys.EVENT_REPLAYER, address(replayer).code);

        /// Reset so its not included in the state dump
        vm.etch(address(replayer), "");
        vm.resetNonce(address(replayer));
    }

    /// @notice Renders the PRIVATE half of a private interop pair.
    /// @dev The private chain is a stock custom gas token chain plus one contract: the
    ///      NativeMintBridge, authorized as a LiquidityController minter so that ETH locked on the
    ///      counterparty can be minted here as native asset. The protocol ETH path stays closed --
    ///      `SuperchainETHBridge` would burn the custom unit while asking a counterparty to mint
    ///      real ETH -- so this bridge is the only way ETH-denominated value crosses the boundary.
    function setPrivateInteropPrivateChain(Input memory _input) internal {
        removePrivateInteropProtocolETHPath();
        setNativeMintBridge(_input);

        // Authorizing the minter is an owner-only call on the LiquidityController proxy, which
        // L2CM initialized with this owner a moment ago.
        vm.prank(_input.liquidityControllerOwner);
        ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER).authorizeMinter(Predeploys.NATIVE_MINT_BRIDGE);
    }

    /// @notice Removes the stock ETH bridge and its unbounded native-liquidity pool from the
    ///         private custom-gas-token half. L2ContractsManager installs both for every ordinary
    ///         interop chain before this private-half specialization runs; leaving them installed
    ///         would create a second, unbacked path around NativeMintBridge and ETHLockVault.
    function removePrivateInteropProtocolETHPath() internal {
        address bridgeImpl = Predeploys.predeployToCodeNamespace(Predeploys.SUPERCHAIN_ETH_BRIDGE);
        address liquidityImpl = Predeploys.predeployToCodeNamespace(Predeploys.ETH_LIQUIDITY);

        EIP1967Helper.setImplementation(Predeploys.SUPERCHAIN_ETH_BRIDGE, address(0));
        vm.etch(bridgeImpl, "");
        EIP1967Helper.setImplementation(Predeploys.ETH_LIQUIDITY, address(0));
        vm.etch(liquidityImpl, "");
        vm.deal(Predeploys.ETH_LIQUIDITY, 0);
        vm.deal(liquidityImpl, 0);
    }

    /// @notice This predeploy is following the safety invariant #2: the counterparty chain ID and
    ///         the counterparty's lock vault address are immutables.
    function setNativeMintBridge(Input memory _input) internal {
        INativeMintBridge bridge = INativeMintBridge(
            DeployUtils.create1({
                _name: "NativeMintBridge",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        INativeMintBridge.__constructor__,
                        (_input.privateInteropCounterpartyChainID, _input.privateInteropLockVault)
                    )
                )
            })
        );
        _setPrivateInteropImplementation(Predeploys.NATIVE_MINT_BRIDGE, address(bridge).code);

        /// Reset so its not included in the state dump
        vm.etch(address(bridge), "");
        vm.resetNonce(address(bridge));
    }

    /// @notice Sets a private interop predeploy's implementation code and points its proxy at it.
    /// @dev The three private interop addresses sit outside the predeploy registry (Predeploys.sol
    ///      explains why), so the registry-driven `setPredeployProxies` etches their Proxy but
    ///      leaves the implementation slot empty, and `_setImplementationCode` cannot look their
    ///      names up. This does both halves explicitly, matching what the registry path produces
    ///      for every other proxied predeploy: implementation code at the code-namespace
    ///      counterpart, admin slot set on both, implementation slot set on the proxy.
    function _setPrivateInteropImplementation(
        address _proxy,
        bytes memory _deployedCode
    )
        internal
        returns (address impl_)
    {
        impl_ = Predeploys.predeployToCodeNamespace(_proxy);
        vm.etch(impl_, _deployedCode);
        EIP1967Helper.setAdmin(impl_, Predeploys.PROXY_ADMIN);
        EIP1967Helper.setImplementation(_proxy, impl_);
    }

    /// @notice Sets all the preinstalls.
    function setPreinstalls() internal {
        address tmpSetPreinstalls = address(uint160(uint256(keccak256("SetPreinstalls"))));
        vm.etch(tmpSetPreinstalls, vm.getDeployedCode("SetPreinstalls.s.sol:SetPreinstalls"));
        SetPreinstalls(tmpSetPreinstalls).setPreinstalls();
        vm.etch(tmpSetPreinstalls, "");
    }

    /// @notice Activate Ecotone network upgrade.
    function activateEcotone() internal {
        require(Preinstalls.BeaconBlockRoots.code.length > 0, "L2Genesis: must have beacon-block-roots contract");
        vm.prank(IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).DEPOSITOR_ACCOUNT());
        IGasPriceOracle(Predeploys.GAS_PRICE_ORACLE).setEcotone();
    }

    function activateFjord() internal {
        vm.prank(IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).DEPOSITOR_ACCOUNT());
        IGasPriceOracle(Predeploys.GAS_PRICE_ORACLE).setFjord();
    }

    function activateIsthmus() internal {
        vm.prank(IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).DEPOSITOR_ACCOUNT());
        IGasPriceOracle(Predeploys.GAS_PRICE_ORACLE).setIsthmus();
    }

    function activateJovian() internal {
        vm.prank(IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).DEPOSITOR_ACCOUNT());
        IGasPriceOracle(Predeploys.GAS_PRICE_ORACLE).setJovian();
    }

    /// @notice Sets the bytecode in state
    function _setImplementationCode(address _addr) internal returns (address) {
        string memory cname = Predeploys.getName(_addr);
        address impl = Predeploys.predeployToCodeNamespace(_addr);
        vm.etch(impl, DeployUtils.getDeployedCode(cname));
        // Set the EIP-1967 admin slot on the implementation so that ProxyAdminOwnedBase.proxyAdmin()
        // can resolve the proxy admin when initialize() is called directly on the implementation.
        EIP1967Helper.setAdmin(impl, Predeploys.PROXY_ADMIN);
        return impl;
    }

    /// @notice Helper function to set up a fee vault predeploy.
    ///         This follows safety invariant #1.
    /// @param _vaultAddr The predeploy address of the fee vault.
    /// @param _useCustomGasToken Whether the chain uses a custom gas token.
    /// @param _withdrawalNetwork The withdrawal network.
    function _setFeeVault(
        address _vaultAddr,
        bool _useCustomGasToken,
        Types.WithdrawalNetwork _withdrawalNetwork
    )
        internal
    {
        if (_useCustomGasToken && _withdrawalNetwork == Types.WithdrawalNetwork.L1) {
            revert("FeeVault: withdrawalNetwork type cannot be L1 when custom gas token is enabled");
        }

        address impl = _setImplementationCode(_vaultAddr);

        /// Initialize the implementation using max value for min withdrawal amount to make it unusable
        IFeeVault(payable(impl)).initialize(address(0), type(uint256).max, Types.WithdrawalNetwork.L1);
    }

    /// @notice Funds the default dev accounts with ether
    function fundDevAccounts() internal {
        for (uint256 i; i < devAccounts.length; i++) {
            vm.deal(devAccounts[i], DEV_ACCOUNT_FUND_AMT);
        }
    }
}

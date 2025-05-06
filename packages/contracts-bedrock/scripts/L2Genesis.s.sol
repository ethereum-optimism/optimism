// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { Script } from "forge-std/Script.sol";
import { OutputMode, OutputModeUtils, Fork, ForkUtils } from "scripts/libraries/Config.sol";
import { SetPreinstalls } from "scripts/SetPreinstalls.s.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { ISequencerFeeVault } from "interfaces/L2/ISequencerFeeVault.sol";
import { IBaseFeeVault } from "interfaces/L2/IBaseFeeVault.sol";
import { IL1FeeVault } from "interfaces/L2/IL1FeeVault.sol";
import { IOperatorFeeVault } from "interfaces/L2/IOperatorFeeVault.sol";
import { IOptimismMintableERC721Factory } from "interfaces/L2/IOptimismMintableERC721Factory.sol";
import { IGovernanceToken } from "interfaces/governance/IGovernanceToken.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IL2StandardBridge } from "interfaces/L2/IL2StandardBridge.sol";
import { IL2ERC721Bridge } from "interfaces/L2/IL2ERC721Bridge.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL2CrossDomainMessenger } from "interfaces/L2/IL2CrossDomainMessenger.sol";
import { IGasPriceOracle } from "interfaces/L2/IGasPriceOracle.sol";
import { IL1Block } from "interfaces/L2/IL1Block.sol";

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
        address governanceTokenOwner;
        uint256 fork;
        bool useInterop;
        bool enableGovernance;
        bool fundDevAccounts;
    }

    using ForkUtils for Fork;
    using OutputModeUtils for OutputMode;

    uint256 internal constant PRECOMPILE_COUNT = 256;

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

    /// @notice Alias for `runWithStateDump` so that no `--sig` needs to be specified.
    function run(Input memory _input) public {
        address deployer = makeAddr("deployer");
        vm.startPrank(deployer);
        vm.chainId(_input.l2ChainID);

        dealEthToPrecompiles();
        setPredeployProxies(_input);
        setPredeployImplementations(_input);
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

        if (forkEquals(_fork, Fork.JOVIAN)) {
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
    ///         the 1967 admin slot set to the ProxyAdmin predeploy. All defined predeploys
    ///         should have their implementations set.
    ///         Warning: the predeploy accounts have contract code, but 0 nonce value, contrary
    ///         to the expected nonce of 1 per EIP-161. This is because the legacy go genesis
    //          script didn't set the nonce and we didn't want to change that behavior when
    ///         migrating genesis generation to Solidity.
    function setPredeployProxies(Input memory _input) internal {
        bytes memory code = vm.getDeployedCode("Proxy.sol:Proxy");
        uint160 prefix = uint160(0x420) << 148;

        for (uint256 i = 0; i < Predeploys.PREDEPLOY_COUNT; i++) {
            address addr = address(prefix | uint160(i));
            if (Predeploys.notProxied(addr)) {
                continue;
            }

            vm.etch(addr, code);
            EIP1967Helper.setAdmin(addr, Predeploys.PROXY_ADMIN);

            if (Predeploys.isSupportedPredeploy(addr, _input.useInterop)) {
                address implementation = Predeploys.predeployToCodeNamespace(addr);
                EIP1967Helper.setImplementation(addr, implementation);
            }
        }
    }

    /// @notice Sets all the implementations for the predeploy proxies. For contracts without proxies,
    ///      sets the deployed bytecode at their expected predeploy address.
    ///      LEGACY_ERC20_ETH and L1_MESSAGE_SENDER are deprecated and are not set.
    function setPredeployImplementations(Input memory _input) internal {
        setLegacyMessagePasser(); // 0
        // 01: legacy, not used in OP-Stack
        setDeployerWhitelist(); // 2
        // 3,4,5: legacy, not used in OP-Stack.
        setWETH(); // 6: WETH (not behind a proxy)
        setL2CrossDomainMessenger(_input.l1CrossDomainMessengerProxy); // 7
        // 8,9,A,B,C,D,E: legacy, not used in OP-Stack.
        setGasPriceOracle(); // f
        setL2StandardBridge(_input.l1StandardBridgeProxy); // 10
        setSequencerFeeVault(_input); // 11
        setOptimismMintableERC20Factory(); // 12
        setL1BlockNumber(); // 13
        setL2ERC721Bridge(_input.l1ERC721BridgeProxy); // 14
        setL1Block(); // 15
        setL2ToL1MessagePasser(); // 16
        setOptimismMintableERC721Factory(_input); // 17
        setProxyAdmin(_input); // 18
        setBaseFeeVault(_input); // 19
        setL1FeeVault(_input); // 1A
        setOperatorFeeVault(); // 1B
        // 1C,1D,1E,1F: not used.
        setSchemaRegistry(); // 20
        setEAS(); // 21
        setGovernanceToken(_input); // 42: OP (not behind a proxy)
        if (_input.useInterop) {
            setCrossL2Inbox(); // 22
            setL2ToL2CrossDomainMessenger(); // 23
            setSuperchainETHBridge(); // 24
            setETHLiquidity(); // 25
            setSuperchainTokenBridge(); // 28
        }
    }

    function setProxyAdmin(Input memory _input) internal {
        // Note the ProxyAdmin implementation itself is behind a proxy that owns itself.
        address impl = _setImplementationCode(Predeploys.PROXY_ADMIN);

        bytes32 _ownerSlot = bytes32(0);

        // there is no initialize() function, so we just set the storage manually.
        vm.store(Predeploys.PROXY_ADMIN, _ownerSlot, bytes32(uint256(uint160(_input.opChainProxyAdminOwner))));
        // update the proxy to not be uninitialized (although not standard initialize pattern)
        vm.store(impl, _ownerSlot, bytes32(uint256(uint160(_input.opChainProxyAdminOwner))));
    }

    function setL2ToL1MessagePasser() internal {
        _setImplementationCode(Predeploys.L2_TO_L1_MESSAGE_PASSER);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL2CrossDomainMessenger(address payable _l1CrossDomainMessengerProxy) internal {
        address impl = _setImplementationCode(Predeploys.L2_CROSS_DOMAIN_MESSENGER);

        IL2CrossDomainMessenger(impl).initialize({ _l1CrossDomainMessenger: ICrossDomainMessenger(address(0)) });

        IL2CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).initialize({
            _l1CrossDomainMessenger: ICrossDomainMessenger(_l1CrossDomainMessengerProxy)
        });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL2StandardBridge(address payable _l1StandardBridgeProxy) internal {
        address impl = _setImplementationCode(Predeploys.L2_STANDARD_BRIDGE);

        IL2StandardBridge(payable(impl)).initialize({ _otherBridge: IStandardBridge(payable(address(0))) });

        IL2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).initialize({
            _otherBridge: IStandardBridge(_l1StandardBridgeProxy)
        });
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL2ERC721Bridge(address payable _l1ERC721BridgeProxy) internal {
        address impl = _setImplementationCode(Predeploys.L2_ERC721_BRIDGE);

        IL2ERC721Bridge(impl).initialize({ _l1ERC721Bridge: payable(address(0)) });

        IL2ERC721Bridge(Predeploys.L2_ERC721_BRIDGE).initialize({ _l1ERC721Bridge: payable(_l1ERC721BridgeProxy) });
    }

    /// @notice This predeploy is following the safety invariant #2,
    function setSequencerFeeVault(Input memory _input) internal {
        ISequencerFeeVault vault = ISequencerFeeVault(
            DeployUtils.create1({
                _name: "SequencerFeeVault",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        ISequencerFeeVault.__constructor__,
                        (
                            _input.sequencerFeeVaultRecipient,
                            _input.sequencerFeeVaultMinimumWithdrawalAmount,
                            Types.WithdrawalNetwork(_input.sequencerFeeVaultWithdrawalNetwork)
                        )
                    )
                )
            })
        );

        address impl = Predeploys.predeployToCodeNamespace(Predeploys.SEQUENCER_FEE_WALLET);
        vm.etch(impl, address(vault).code);

        /// Reset so its not included state dump
        vm.etch(address(vault), "");
        vm.resetNonce(address(vault));
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setOptimismMintableERC20Factory() internal {
        address impl = _setImplementationCode(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY);

        IOptimismMintableERC20Factory(impl).initialize({ _bridge: address(0) });

        IOptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY).initialize({
            _bridge: Predeploys.L2_STANDARD_BRIDGE
        });
    }

    /// @notice This predeploy is following the safety invariant #2,
    function setOptimismMintableERC721Factory(Input memory _input) internal {
        IOptimismMintableERC721Factory factory = IOptimismMintableERC721Factory(
            DeployUtils.create1({
                _name: "OptimismMintableERC721Factory",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IOptimismMintableERC721Factory.__constructor__, (Predeploys.L2_ERC721_BRIDGE, _input.l1ChainID)
                    )
                )
            })
        );

        address impl = Predeploys.predeployToCodeNamespace(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY);
        vm.etch(impl, address(factory).code);

        /// Reset so its not included state dump
        vm.etch(address(factory), "");
        vm.resetNonce(address(factory));
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL1Block() internal {
        // Note: L1 block attributes are set to 0.
        // Before the first user-tx the state is overwritten with actual L1 attributes.
        _setImplementationCode(Predeploys.L1_BLOCK_ATTRIBUTES);
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
        vm.etch(Predeploys.WETH, vm.getDeployedCode("WETH.sol:WETH"));
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setL1BlockNumber() internal {
        _setImplementationCode(Predeploys.L1_BLOCK_NUMBER);
    }

    /// @notice This predeploy is following the safety invariant #1.
    function setLegacyMessagePasser() internal {
        _setImplementationCode(Predeploys.LEGACY_MESSAGE_PASSER);
    }

    /// @notice This predeploy is following the safety invariant #2.
    function setBaseFeeVault(Input memory _input) internal {
        IBaseFeeVault vault = IBaseFeeVault(
            DeployUtils.create1({
                _name: "BaseFeeVault",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IBaseFeeVault.__constructor__,
                        (
                            _input.baseFeeVaultRecipient,
                            _input.baseFeeVaultMinimumWithdrawalAmount,
                            Types.WithdrawalNetwork(_input.baseFeeVaultWithdrawalNetwork)
                        )
                    )
                )
            })
        );

        address impl = Predeploys.predeployToCodeNamespace(Predeploys.BASE_FEE_VAULT);
        vm.etch(impl, address(vault).code);

        /// Reset so its not included state dump
        vm.etch(address(vault), "");
        vm.resetNonce(address(vault));
    }

    /// @notice This predeploy is following the safety invariant #2.
    function setL1FeeVault(Input memory _input) internal {
        IL1FeeVault vault = IL1FeeVault(
            DeployUtils.create1({
                _name: "L1FeeVault",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IL1FeeVault.__constructor__,
                        (
                            _input.l1FeeVaultRecipient,
                            _input.l1FeeVaultMinimumWithdrawalAmount,
                            Types.WithdrawalNetwork(_input.l1FeeVaultWithdrawalNetwork)
                        )
                    )
                )
            })
        );

        address impl = Predeploys.predeployToCodeNamespace(Predeploys.L1_FEE_VAULT);
        vm.etch(impl, address(vault).code);

        /// Reset so its not included state dump
        vm.etch(address(vault), "");
        vm.resetNonce(address(vault));
    }

    /// @notice This predeploy is following the safety invariant #2.
    function setOperatorFeeVault() internal {
        IOperatorFeeVault vault = IOperatorFeeVault(
            DeployUtils.create1({
                _name: "OperatorFeeVault",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IOperatorFeeVault.__constructor__, ()))
            })
        );

        address impl = Predeploys.predeployToCodeNamespace(Predeploys.OPERATOR_FEE_VAULT);
        vm.etch(impl, address(vault).code);

        /// Reset so its not included state dump
        vm.etch(address(vault), "");
        vm.resetNonce(address(vault));
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
        bytes memory code = vm.getCode(string.concat(cname, ".sol:", cname));

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
        _setImplementationCode(Predeploys.CROSS_L2_INBOX);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setL2ToL2CrossDomainMessenger() internal {
        _setImplementationCode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setETHLiquidity() internal {
        _setImplementationCode(Predeploys.ETH_LIQUIDITY);
        vm.deal(Predeploys.ETH_LIQUIDITY, type(uint248).max);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setSuperchainETHBridge() internal {
        _setImplementationCode(Predeploys.SUPERCHAIN_ETH_BRIDGE);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setOptimismSuperchainERC20Factory() internal {
        _setImplementationCode(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setOptimismSuperchainERC20Beacon() internal {
        address superchainERC20Impl = Predeploys.OPTIMISM_SUPERCHAIN_ERC20;
        vm.etch(superchainERC20Impl, vm.getDeployedCode("OptimismSuperchainERC20.sol:OptimismSuperchainERC20"));

        _setImplementationCode(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON);
    }

    /// @notice This predeploy is following the safety invariant #1.
    ///         This contract has no initializer.
    function setSuperchainTokenBridge() internal {
        _setImplementationCode(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
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

    /// @notice Sets the bytecode in state
    function _setImplementationCode(address _addr) internal returns (address) {
        string memory cname = Predeploys.getName(_addr);
        address impl = Predeploys.predeployToCodeNamespace(_addr);
        vm.etch(impl, vm.getDeployedCode(string.concat(cname, ".sol:", cname)));
        return impl;
    }

    /// @notice Funds the default dev accounts with ether
    function fundDevAccounts() internal {
        for (uint256 i; i < devAccounts.length; i++) {
            vm.deal(devAccounts[i], DEV_ACCOUNT_FUND_AMT);
        }
    }
}

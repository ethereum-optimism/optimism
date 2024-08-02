// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { Vm, VmSafe } from "forge-std/Vm.sol";

import { Deployer } from "scripts/Deployer.sol";

import { Proxy } from "src/universal/Proxy.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";
import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { Chains } from "scripts/Chains.sol";
import { Config } from "scripts/Config.sol";

import { IBigStepper } from "src/dispute/interfaces/IBigStepper.sol";
import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";
import "src/dispute/lib/Types.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { ChainAssertions } from "scripts/ChainAssertions.sol";
import { Types } from "scripts/Types.sol";

/// @title Deploy
/// @notice Script used to upgrade the FP contracts
contract Deploy is Deployer {
    using stdJson for string;

    /// @notice FaultDisputeGameParams is a struct that contains the parameters necessary to call
    ///         the function _setFaultGameImplementation. This struct exists because the EVM needs
    ///         to finally adopt PUSHN and get rid of stack too deep once and for all.
    ///         Someday we will look back and laugh about stack too deep, today is not that day.
    struct FaultDisputeGameParams {
        AnchorStateRegistry anchorStateRegistry;
        DelayedWETH weth;
        GameType gameType;
        Claim absolutePrestate;
        IBigStepper faultVm;
        uint256 maxGameDepth;
    }

    ////////////////////////////////////////////////////////////////
    //                         Helper                            //
    ////////////////////////////////////////////////////////////////

    /// @notice read proxyd address from the hardhat deployment files
    function readProxyAddress(string memory _contractName) internal returns (address _proxyAddress) {
        string memory _hardhatDeploymentPath = vm.envOr("HARDHAT_DEPLOYMENT_PATH", string(""));
        require(
            bytes(_hardhatDeploymentPath).length > 0,
            "Deploy: must set HARDHAT_DEPLOYMENT_PATH to filesystem path of hardhat deployment files"
        );
        string memory _contractJson = vm.readFile(_hardhatDeploymentPath);
        bytes memory _contractAddress = stdJson.parseRaw(_contractJson, string(abi.encodePacked(".", _contractName)));
        _proxyAddress = bytesToAddress(_contractAddress);
    }

    /// @notice Convert bytes to address
    function bytesToAddress(bytes memory bys) private pure returns (address addr) {
        assembly {
            addr := mload(add(bys, 32))
        }
    }

    ////////////////////////////////////////////////////////////////
    //                        Modifiers                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Modifier that wraps a function in broadcasting.
    modifier broadcast() {
        vm.startBroadcast(msg.sender);
        _;
        vm.stopBroadcast();
    }

    ////////////////////////////////////////////////////////////////
    //                        Accessors                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Returns the proxy addresses. If a proxy is not found, it will have address(0).
    function _proxies() internal view returns (Types.ContractSet memory proxies_) {
        proxies_ = Types.ContractSet({
            L1CrossDomainMessenger: mustGetAddress("L1CrossDomainMessengerProxy"),
            L1StandardBridge: mustGetAddress("L1StandardBridgeProxy"),
            L2OutputOracle: mustGetAddress("L2OutputOracleProxy"),
            DisputeGameFactory: mustGetAddress("DisputeGameFactoryProxy"),
            DelayedWETH: mustGetAddress("DelayedWETHProxy"),
            AnchorStateRegistry: mustGetAddress("AnchorStateRegistryProxy"),
            OptimismMintableERC20Factory: mustGetAddress("OptimismMintableERC20FactoryProxy"),
            OptimismPortal: mustGetAddress("OptimismPortalProxy"),
            OptimismPortal2: mustGetAddress("OptimismPortalProxy"),
            SystemConfig: mustGetAddress("SystemConfigProxy"),
            L1ERC721Bridge: mustGetAddress("L1ERC721BridgeProxy"),
            ProtocolVersions: mustGetAddress("ProtocolVersionsProxy"),
            SuperchainConfig: mustGetAddress("SuperchainConfigProxy")
        });
    }

    /// @notice Returns the proxy addresses, not reverting if any are unset.
    function _proxiesUnstrict() internal view returns (Types.ContractSet memory proxies_) {
        proxies_ = Types.ContractSet({
            L1CrossDomainMessenger: getAddress("L1CrossDomainMessengerProxy"),
            L1StandardBridge: getAddress("L1StandardBridgeProxy"),
            L2OutputOracle: getAddress("L2OutputOracleProxy"),
            DisputeGameFactory: getAddress("DisputeGameFactoryProxy"),
            DelayedWETH: getAddress("DelayedWETHProxy"),
            AnchorStateRegistry: getAddress("AnchorStateRegistryProxy"),
            OptimismMintableERC20Factory: getAddress("OptimismMintableERC20FactoryProxy"),
            OptimismPortal: getAddress("OptimismPortalProxy"),
            OptimismPortal2: getAddress("OptimismPortalProxy"),
            SystemConfig: getAddress("SystemConfigProxy"),
            L1ERC721Bridge: getAddress("L1ERC721BridgeProxy"),
            ProtocolVersions: getAddress("ProtocolVersionsProxy"),
            SuperchainConfig: getAddress("SuperchainConfigProxy")
        });
    }

    ////////////////////////////////////////////////////////////////
    //            State Changing Helper Functions                 //
    ////////////////////////////////////////////////////////////////

    /// @notice Call to the Proxy's upgrade and call method
    function _upgradeToAndCall(address payable _proxy, address _implementation, bytes memory _innerCallData) internal {
        Proxy(_proxy).upgradeToAndCall(_implementation, _innerCallData);
    }

    ////////////////////////////////////////////////////////////////
    //                    SetUp and Run                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Deploy all FP contracts.
    function run() public {
        console.log("Upgrading the protocol to support FP");
        _run();
    }

    /// @notice Internal function containing the deploy logic.
    function _run() internal virtual {
        console.log("Start of L1 Deploy!");
        deployProxies();
        deployImplementations();
        initializeImplementations();

        setAlphabetFaultGameImplementation({ _allowUpgrade: false });
        setCannonFaultGameImplementation({ _allowUpgrade: false });
        setPermissionedCannonFaultGameImplementation({ _allowUpgrade: false });

        transferERC1967Proxy("DisputeGameFactoryProxy");
        transferERC1967Proxy("DelayedWETHProxy");
        transferERC1967Proxy("AnchorStateRegistryProxy");

        console.log("PLEASE UPGRADE OptimismPortal2 MANUALLY");
        console.log("PLEASE UPDATE DGF ADDRESS IN SystemConfigProxy MANUALLY");
    }

    /// @notice Deploy all of the proxies
    function deployProxies() public {
        console.log("Deploying proxies");

        // Both the DisputeGameFactory and L2OutputOracle proxies are deployed regardles of whether FPAC is enabled
        // to prevent a nastier refactor to the deploy scripts. In the future, the L2OutputOracle will be removed. If
        // fault proofs are not enabled, the DisputeGameFactory proxy will be unused.
        deployERC1967ProxyWithOwner("DisputeGameFactoryProxy", msg.sender);
        deployERC1967ProxyWithOwner("DelayedWETHProxy", msg.sender);
        deployERC1967ProxyWithOwner("AnchorStateRegistryProxy", msg.sender);
    }

    /// @notice Deploy all of the implementations
    function deployImplementations() public {
        console.log("Deploying implementations");

        // Fault proofs
        deployOptimismPortal2();
        deployDisputeGameFactory();
        deployDelayedWETH();
        deployPreimageOracle();
        deployMips();
        deployAnchorStateRegistry();
    }

    /// @notice Initialize all of the implementations
    function initializeImplementations() public {
        console.log("Initializing implementations");
        // This has to be manually triggered!
        // because we have already initialized the contracts in the previous deployment
        // initializeOptimismPortal2();

        initializeDisputeGameFactory();
        initializeDelayedWETH();
        initializeAnchorStateRegistry();
    }

    /// @notice Transfer the ERC1967Proxy ownership to the ProxyAdmin
    function transferERC1967Proxy(string memory _name) public {
        address proxyAdmin = readProxyAddress("ProxyAdmin");
        require(proxyAdmin != address(0), "Deploy: ProxyAdmin address not found");
        EIP1967Helper.setAdmin(mustGetAddress(_name), proxyAdmin);

        // verify the admin
        require(EIP1967Helper.getAdmin(mustGetAddress(_name)) == proxyAdmin, "Deploy: ProxyAdmin transfer failed");
        console.log("Transferred %s ownership to ProxyAdmin at %s", _name, proxyAdmin);
    }

    /// @notice Deploys an ERC1967Proxy contract with a specified owner.
    /// @param _name The name of the proxy contract to be deployed.
    /// @param _proxyOwner The address of the owner of the proxy contract.
    /// @return addr_ The address of the deployed proxy contract.
    function deployERC1967ProxyWithOwner(
        string memory _name,
        address _proxyOwner
    )
        public
        broadcast
        returns (address addr_)
    {
        console.log(string.concat("Deploying ERC1967 proxy for ", _name));
        Proxy proxy = new Proxy({ _admin: _proxyOwner });

        require(EIP1967Helper.getAdmin(address(proxy)) == _proxyOwner);

        save(_name, address(proxy));
        console.log("   at %s", address(proxy));
        addr_ = address(proxy);
    }

    /// @notice Deploy the OptimismPortal2
    function deployOptimismPortal2() public broadcast returns (address addr_) {
        console.log("Deploying OptimismPortal2 implementation");

        // Could also verify this inside DeployConfig but doing it here is a bit more reliable.
        require(
            uint32(cfg.respectedGameType()) == cfg.respectedGameType(), "Deploy: respectedGameType must fit into uint32"
        );

        OptimismPortal2 portal = new OptimismPortal2({
            _proofMaturityDelaySeconds: cfg.proofMaturityDelaySeconds(),
            _disputeGameFinalityDelaySeconds: cfg.disputeGameFinalityDelaySeconds()
        });

        save("OptimismPortal2", address(portal));
        console.log("OptimismPortal2 deployed at %s", address(portal));

        // Override the `OptimismPortal2` contract to the deployed implementation. This is necessary
        // to check the `OptimismPortal2` implementation alongside dependent contracts, which
        // are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.OptimismPortal2 = address(portal);
        ChainAssertions.checkOptimismPortal2({ _contracts: contracts, _cfg: cfg, _isProxy: false });

        addr_ = address(portal);
    }

    /// @notice Deploy the DisputeGameFactory
    function deployDisputeGameFactory() public broadcast returns (address addr_) {
        console.log("Deploying DisputeGameFactory implementation");
        DisputeGameFactory factory = new DisputeGameFactory();
        save("DisputeGameFactory", address(factory));
        console.log("DisputeGameFactory deployed at %s", address(factory));

        // Override the `DisputeGameFactory` contract to the deployed implementation. This is necessary to check the
        // `DisputeGameFactory` implementation alongside dependent contracts, which are always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.DisputeGameFactory = address(factory);
        ChainAssertions.checkDisputeGameFactory({ _contracts: contracts, _expectedOwner: address(0) });

        addr_ = address(factory);
    }

    function deployDelayedWETH() public broadcast returns (address addr_) {
        console.log("Deploying DelayedWETH implementation");
        DelayedWETH weth = new DelayedWETH(cfg.faultGameWithdrawalDelay());
        save("DelayedWETH", address(weth));
        console.log("DelayedWETH deployed at %s", address(weth));

        // Override the `DelayedWETH` contract to the deployed implementation. This is necessary
        // to check the `DelayedWETH` implementation alongside dependent contracts, which are
        // always proxies.
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.DelayedWETH = address(weth);
        ChainAssertions.checkDelayedWETH({
            _contracts: contracts,
            _cfg: cfg,
            _isProxy: false,
            _expectedOwner: address(0)
        });

        addr_ = address(weth);
    }

    /// @notice Deploy the PreimageOracle
    function deployPreimageOracle() public broadcast returns (address addr_) {
        console.log("Deploying PreimageOracle implementation");
        PreimageOracle preimageOracle = new PreimageOracle({
            _minProposalSize: cfg.preimageOracleMinProposalSize(),
            _challengePeriod: cfg.preimageOracleChallengePeriod()
        });
        save("PreimageOracle", address(preimageOracle));
        console.log("PreimageOracle deployed at %s", address(preimageOracle));

        addr_ = address(preimageOracle);
    }

    /// @notice Deploy Mips
    function deployMips() public broadcast returns (address addr_) {
        console.log("Deploying Mips implementation");
        MIPS mips = new MIPS(IPreimageOracle(mustGetAddress("PreimageOracle")));
        save("Mips", address(mips));
        console.log("MIPS deployed at %s", address(mips));

        addr_ = address(mips);
    }

    /// @notice Deploy the AnchorStateRegistry
    function deployAnchorStateRegistry() public broadcast returns (address addr_) {
        console.log("Deploying AnchorStateRegistry implementation");
        AnchorStateRegistry anchorStateRegistry =
            new AnchorStateRegistry(DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy")));
        save("AnchorStateRegistry", address(anchorStateRegistry));
        console.log("AnchorStateRegistry deployed at %s", address(anchorStateRegistry));

        addr_ = address(anchorStateRegistry);
    }

    /// @notice Initialize the DisputeGameFactory
    function initializeDisputeGameFactory() public broadcast {
        console.log("Upgrading and initializing DisputeGameFactory proxy");
        address disputeGameFactoryProxy = mustGetAddress("DisputeGameFactoryProxy");
        address disputeGameFactory = mustGetAddress("DisputeGameFactory");

        _upgradeToAndCall({
            _proxy: payable(disputeGameFactoryProxy),
            _implementation: disputeGameFactory,
            _innerCallData: abi.encodeCall(DisputeGameFactory.initialize, (msg.sender))
        });

        string memory version = DisputeGameFactory(disputeGameFactoryProxy).version();
        console.log("DisputeGameFactory version: %s", version);

        ChainAssertions.checkDisputeGameFactory({ _contracts: _proxiesUnstrict(), _expectedOwner: msg.sender });
    }

    function initializeDelayedWETH() public broadcast {
        console.log("Upgrading and initializing DelayedWETH proxy");
        address delayedWETHProxy = mustGetAddress("DelayedWETHProxy");
        address delayedWETH = mustGetAddress("DelayedWETH");
        address superchainConfigProxy = readProxyAddress("SuperchainConfigProxy");
        require(superchainConfigProxy != address(0), "Deploy: SuperchainConfigProxy address not found");
        console.log("SuperchainConfigProxy: ", superchainConfigProxy);

        // Override superchainconfig address
        Types.ContractSet memory contracts = _proxiesUnstrict();
        contracts.SuperchainConfig = address(superchainConfigProxy);

        _upgradeToAndCall({
            _proxy: payable(delayedWETHProxy),
            _implementation: delayedWETH,
            _innerCallData: abi.encodeCall(DelayedWETH.initialize, (msg.sender, SuperchainConfig(superchainConfigProxy)))
        });

        string memory version = DelayedWETH(payable(delayedWETHProxy)).version();
        console.log("DelayedWETH version: %s", version);

        ChainAssertions.checkDelayedWETH({ _contracts: contracts, _cfg: cfg, _isProxy: true, _expectedOwner: msg.sender });
    }

    function initializeAnchorStateRegistry() public broadcast {
        console.log("Upgrading and initializing AnchorStateRegistry proxy");
        address anchorStateRegistryProxy = mustGetAddress("AnchorStateRegistryProxy");
        address anchorStateRegistry = mustGetAddress("AnchorStateRegistry");

        AnchorStateRegistry.StartingAnchorRoot[] memory roots = new AnchorStateRegistry.StartingAnchorRoot[](4);
        roots[0] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.CANNON,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[1] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.PERMISSIONED_CANNON,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[2] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.ALPHABET,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });
        roots[3] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.ASTERISC,
            outputRoot: OutputRoot({
                root: Hash.wrap(cfg.faultGameGenesisOutputRoot()),
                l2BlockNumber: cfg.faultGameGenesisBlock()
            })
        });

        _upgradeToAndCall({
            _proxy: payable(anchorStateRegistryProxy),
            _implementation: anchorStateRegistry,
            _innerCallData: abi.encodeCall(AnchorStateRegistry.initialize, (roots))
        });

        string memory version = AnchorStateRegistry(payable(anchorStateRegistryProxy)).version();
        console.log("AnchorStateRegistry version: %s", version);
    }

    /// @notice Sets the implementation for the `CANNON` game type in the `DisputeGameFactory`
    function setCannonFaultGameImplementation(bool _allowUpgrade) public broadcast {
        console.log("Setting Cannon FaultDisputeGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        DelayedWETH weth = DelayedWETH(mustGetAddress("DelayedWETHProxy"));

        // Set the Cannon FaultDisputeGame implementation in the factory.
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: AnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                weth: weth,
                gameType: GameTypes.CANNON,
                absolutePrestate: loadMipsAbsolutePrestate(),
                faultVm: IBigStepper(mustGetAddress("Mips")),
                maxGameDepth: cfg.faultGameMaxDepth()
            })
        });
    }

    /// @notice Sets the implementation for the `PERMISSIONED_CANNON` game type in the `DisputeGameFactory`
    function setPermissionedCannonFaultGameImplementation(bool _allowUpgrade) public broadcast {
        console.log("Setting Cannon PermissionedDisputeGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        DelayedWETH weth = DelayedWETH(mustGetAddress("DelayedWETHProxy"));

        // Set the Cannon FaultDisputeGame implementation in the factory.
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: AnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                weth: weth,
                gameType: GameTypes.PERMISSIONED_CANNON,
                absolutePrestate: loadMipsAbsolutePrestate(),
                faultVm: IBigStepper(mustGetAddress("Mips")),
                maxGameDepth: cfg.faultGameMaxDepth()
            })
        });
    }

    /// @notice Sets the implementation for the `ALPHABET` game type in the `DisputeGameFactory`
    function setAlphabetFaultGameImplementation(bool _allowUpgrade) public broadcast {
        console.log("Setting Alphabet FaultDisputeGame implementation");
        DisputeGameFactory factory = DisputeGameFactory(mustGetAddress("DisputeGameFactoryProxy"));
        DelayedWETH weth = DelayedWETH(mustGetAddress("DelayedWETHProxy"));

        Claim outputAbsolutePrestate = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        _setFaultGameImplementation({
            _factory: factory,
            _allowUpgrade: _allowUpgrade,
            _params: FaultDisputeGameParams({
                anchorStateRegistry: AnchorStateRegistry(mustGetAddress("AnchorStateRegistryProxy")),
                weth: weth,
                gameType: GameTypes.ALPHABET,
                absolutePrestate: outputAbsolutePrestate,
                faultVm: IBigStepper(new AlphabetVM(outputAbsolutePrestate, PreimageOracle(mustGetAddress("PreimageOracle")))),
                // The max depth for the alphabet trace is always 3. Add 1 because split depth is fully inclusive.
                maxGameDepth: cfg.faultGameSplitDepth() + 3 + 1
            })
        });
    }

    /// @notice Sets the implementation for the given fault game type in the `DisputeGameFactory`.
    function _setFaultGameImplementation(
        DisputeGameFactory _factory,
        bool _allowUpgrade,
        FaultDisputeGameParams memory _params
    )
        internal
    {
        if (address(_factory.gameImpls(_params.gameType)) != address(0) && !_allowUpgrade) {
            console.log(
                "[WARN] DisputeGameFactoryProxy: `FaultDisputeGame` implementation already set for game type: %s",
                vm.toString(GameType.unwrap(_params.gameType))
            );
            return;
        }

        uint32 rawGameType = GameType.unwrap(_params.gameType);
        if (rawGameType != GameTypes.PERMISSIONED_CANNON.raw()) {
            _factory.setImplementation(
                _params.gameType,
                new FaultDisputeGame({
                    _gameType: _params.gameType,
                    _absolutePrestate: _params.absolutePrestate,
                    _maxGameDepth: _params.maxGameDepth,
                    _splitDepth: cfg.faultGameSplitDepth(),
                    _clockExtension: Duration.wrap(uint64(cfg.faultGameClockExtension())),
                    _maxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration())),
                    _vm: _params.faultVm,
                    _weth: _params.weth,
                    _anchorStateRegistry: _params.anchorStateRegistry,
                    _l2ChainId: cfg.l2ChainID()
                })
            );
        } else {
            _factory.setImplementation(
                _params.gameType,
                new PermissionedDisputeGame({
                    _gameType: _params.gameType,
                    _absolutePrestate: _params.absolutePrestate,
                    _maxGameDepth: _params.maxGameDepth,
                    _splitDepth: cfg.faultGameSplitDepth(),
                    _clockExtension: Duration.wrap(uint64(cfg.faultGameClockExtension())),
                    _maxClockDuration: Duration.wrap(uint64(cfg.faultGameMaxClockDuration())),
                    _vm: _params.faultVm,
                    _weth: _params.weth,
                    _anchorStateRegistry: _params.anchorStateRegistry,
                    _l2ChainId: cfg.l2ChainID(),
                    _proposer: cfg.l2OutputOracleProposer(),
                    _challenger: cfg.l2OutputOracleChallenger()
                })
            );
        }

        string memory gameTypeString;
        if (rawGameType == GameTypes.CANNON.raw()) {
            gameTypeString = "Cannon";
        } else if (rawGameType == GameTypes.PERMISSIONED_CANNON.raw()) {
            gameTypeString = "PermissionedCannon";
        } else if (rawGameType == GameTypes.ALPHABET.raw()) {
            gameTypeString = "Alphabet";
        } else {
            gameTypeString = "Unknown";
        }

        console.log(
            "DisputeGameFactoryProxy: set `FaultDisputeGame` implementation (Backend: %s | GameType: %s)",
            gameTypeString,
            vm.toString(rawGameType)
        );
    }

    /// @notice Loads the mips absolute prestate from the prestate-proof for devnets otherwise
    ///         from the config.
    function loadMipsAbsolutePrestate() internal returns (Claim mipsAbsolutePrestate_) {
        if (block.chainid == Chains.LocalDevnet || block.chainid == Chains.GethDevnet) {
            // Fetch the absolute prestate dump
            string memory filePath = string.concat(vm.projectRoot(), "/../../op-program/bin/prestate-proof.json");
            string[] memory commands = new string[](3);
            commands[0] = "bash";
            commands[1] = "-c";
            commands[2] = string.concat("[[ -f ", filePath, " ]] && echo \"present\"");
            if (vm.ffi(commands).length == 0) {
                revert("Cannon prestate dump not found, generate it with `make cannon-prestate` in the monorepo root.");
            }
            commands[2] = string.concat("cat ", filePath, " | jq -r .pre");
            mipsAbsolutePrestate_ = Claim.wrap(abi.decode(vm.ffi(commands), (bytes32)));
            console.log(
                "[Cannon Dispute Game] Using devnet MIPS Absolute prestate: %s",
                vm.toString(Claim.unwrap(mipsAbsolutePrestate_))
            );
        } else {
            console.log(
                "[Cannon Dispute Game] Using absolute prestate from config: %x", cfg.faultGameAbsolutePrestate()
            );
            mipsAbsolutePrestate_ = Claim.wrap(bytes32(cfg.faultGameAbsolutePrestate()));
        }
    }
}

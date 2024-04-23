// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { console2 as console } from "forge-std/console2.sol";

import { Script } from "forge-std/Script.sol";
import { Artifacts } from "scripts/Artifacts.s.sol";
import { PeripheryDeployConfig } from "scripts/PeripheryDeployConfig.s.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";

import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";

import { Faucet } from "src/periphery/faucet/Faucet.sol";
import { Drippie } from "src/periphery/drippie/Drippie.sol";
import { CheckGelatoLow } from "src/periphery/drippie/dripchecks/CheckGelatoLow.sol";
import { CheckBalanceLow } from "src/periphery/drippie/dripchecks/CheckBalanceLow.sol";
import { CheckTrue } from "src/periphery/drippie/dripchecks/CheckTrue.sol";
import { AdminFaucetAuthModule } from "src/periphery/faucet/authmodules/AdminFaucetAuthModule.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { Config } from "scripts/Config.sol";

/// @title DeployPeriphery
/// @notice Script used to deploy periphery contracts.
contract DeployPeriphery is Script, Artifacts {
    PeripheryDeployConfig cfg;

    /// @notice The name of the script, used to ensure the right deploy artifacts are used.
    function name() public pure returns (string memory name_) {
        name_ = "DeployPeriphery";
    }

    /// @notice Sets up the deployment script.
    function setUp() public override {
        Artifacts.setUp();
        string memory path = string.concat(vm.projectRoot(), "/periphery-deploy-config/", deploymentContext, ".json");
        cfg = new PeripheryDeployConfig(path);
        console.log("Deployment context: %s", deploymentContext);
    }

    /// @notice Deploy all of the periphery contracts.
    function run() public {
        console.log("Deploying all periphery contracts");

        deployProxies();
        deployImplementations();

        initializeFaucet();
        installFaucetAuthModulesConfigs();

        if (cfg.installOpChainFaucetsDrips()) {
            installOpChainFaucetsDrippieConfigs();
        }

        if (cfg.archivePreviousOpChainFaucetsDrips()) {
            archivePreviousOpChainFaucetsDrippieConfigs();
        }
    }

    /// @notice Deploy all of the proxies.
    function deployProxies() public {
        deployProxyAdmin();
        deployFaucetProxy();
    }

    /// @notice Deploy all of the implementations.
    function deployImplementations() public {
        deployFaucet();
        deployFaucetDrippie();
        deployCheckTrue();
        deployCheckBalanceLow();
        deployCheckGelatoLow();
        deployOnChainAuthModule();
        deployOffChainAuthModule();
    }

    /// @notice Modifier that wraps a function in broadcasting.
    modifier broadcast() {
        vm.startBroadcast();
        _;
        vm.stopBroadcast();
    }

    /// @notice Deploy ProxyAdmin.
    function deployProxyAdmin() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "ProxyAdmin",
            _creationCode: type(ProxyAdmin).creationCode,
            _constructorParams: abi.encode(msg.sender)
        });

        ProxyAdmin admin = ProxyAdmin(addr_);
        require(admin.owner() == msg.sender);
    }

    /// @notice Deploy FaucetProxy.
    function deployFaucetProxy() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "FaucetProxy",
            _creationCode: type(Proxy).creationCode,
            _constructorParams: abi.encode(mustGetAddress("ProxyAdmin"))
        });

        Proxy proxy = Proxy(payable(addr_));
        require(EIP1967Helper.getAdmin(address(proxy)) == mustGetAddress("ProxyAdmin"));
    }

    /// @notice Deploy the Faucet contract.
    function deployFaucet() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "Faucet",
            _creationCode: type(Faucet).creationCode,
            _constructorParams: abi.encode(cfg.faucetAdmin())
        });

        Faucet faucet = Faucet(payable(addr_));
        require(faucet.ADMIN() == cfg.faucetAdmin());
    }

    /// @notice Deploy the Drippie contract.
    function deployFaucetDrippie() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "FaucetDrippie",
            _creationCode: type(Drippie).creationCode,
            _constructorParams: abi.encode(cfg.faucetDrippieOwner())
        });

        Drippie drippie = Drippie(payable(addr_));
        require(drippie.owner() == cfg.faucetDrippieOwner());
    }

    /// @notice Deploy On-Chain Authentication Module.
    function deployOnChainAuthModule() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "OnChainAuthModule",
            _creationCode: type(AdminFaucetAuthModule).creationCode,
            _constructorParams: abi.encode(cfg.faucetOnchainAuthModuleAdmin(), "OnChainAuthModule", "1")
        });

        AdminFaucetAuthModule module = AdminFaucetAuthModule(addr_);
        require(module.ADMIN() == cfg.faucetOnchainAuthModuleAdmin());
    }

    /// @notice Deploy Off-Chain Authentication Module.
    function deployOffChainAuthModule() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "OffChainAuthModule",
            _creationCode: type(AdminFaucetAuthModule).creationCode,
            _constructorParams: abi.encode(cfg.faucetOffchainAuthModuleAdmin(), "OffChainAuthModule", "1")
        });

        AdminFaucetAuthModule module = AdminFaucetAuthModule(addr_);
        require(module.ADMIN() == cfg.faucetOffchainAuthModuleAdmin());
    }

    /// @notice Deploy CheckTrue contract.
    function deployCheckTrue() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "CheckTrue",
            _creationCode: type(CheckTrue).creationCode,
            _constructorParams: hex""
        });
    }

    /// @notice Deploy CheckBalanceLow contract.
    function deployCheckBalanceLow() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "CheckBalanceLow",
            _creationCode: type(CheckBalanceLow).creationCode,
            _constructorParams: hex""
        });
    }

    /// @notice Deploy CheckGelatoLow contract.
    function deployCheckGelatoLow() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "CheckGelatoLow",
            _creationCode: type(CheckGelatoLow).creationCode,
            _constructorParams: hex""
        });
    }

    /// @notice Initialize the Faucet.
    function initializeFaucet() public broadcast {
        ProxyAdmin proxyAdmin = ProxyAdmin(mustGetAddress("ProxyAdmin"));
        address faucetProxy = mustGetAddress("FaucetProxy");
        address faucet = mustGetAddress("Faucet");
        address implementationAddress = proxyAdmin.getProxyImplementation(faucetProxy);
        if (implementationAddress == faucet) {
            console.log("Faucet proxy implementation already set");
        } else {
            proxyAdmin.upgrade({ _proxy: payable(faucetProxy), _implementation: faucet });
        }

        require(Faucet(payable(faucetProxy)).ADMIN() == Faucet(payable(faucet)).ADMIN());
    }

    /// @notice Installs the drip configs in the faucet drippie contract.
    function installFaucetDrippieConfigs() public {
        Drippie drippie = Drippie(mustGetAddress("FaucetDrippie"));
        console.log("Installing faucet drips at %s", address(drippie));
        installFaucetDripV1();
        installFaucetDripV2();
        installFaucetAdminDripV1();
        installFaucetGelatoBalanceV2();
        console.log("Faucet drip configs successfully installed");
    }

    /// @notice Installs drip configs that deposit funds to all OP Chain faucets. This function
    ///         should only be called on an L1 testnet.
    function installOpChainFaucetsDrippieConfigs() public {
        uint256 drippieOwnerPrivateKey = Config.drippieOwnerPrivateKey();
        vm.startBroadcast(drippieOwnerPrivateKey);

        Drippie drippie = Drippie(mustGetAddress("FaucetDrippie"));
        console.log("Installing OP Chain faucet drips at %s", address(drippie));
        installSmallOpChainFaucetsDrips();
        installLargeOpChainFaucetsDrips();
        installSmallOpChainAdminWalletDrips();
        installLargeOpChainAdminWalletDrips();
        console.log("OP chain faucet drip configs successfully installed");

        vm.stopBroadcast();
    }

    /// @notice Installs drips that send funds to small OP chain faucets on the scheduled interval.
    function installSmallOpChainFaucetsDrips() public {
        for (uint256 i = 0; i < cfg.getSmallFaucetsL1BridgeAddressesCount(); i++) {
            address l1BridgeAddress = cfg.smallFaucetsL1BridgeAddresses(i);
            _installDepositEthToDrip({
                _drippie: Drippie(mustGetAddress("FaucetDrippie")),
                _name: _makeFaucetDripName(l1BridgeAddress, cfg.dripVersion()),
                _bridge: l1BridgeAddress,
                _target: mustGetAddress("FaucetProxy"),
                _value: cfg.smallOpChainFaucetDripValue(),
                _interval: cfg.smallOpChainFaucetDripInterval()
            });
        }
    }

    /// @notice Installs drips that send funds to large OP chain faucets on the scheduled interval.
    function installLargeOpChainFaucetsDrips() public {
        for (uint256 i = 0; i < cfg.getLargeFaucetsL1BridgeAddressesCount(); i++) {
            address l1BridgeAddress = cfg.largeFaucetsL1BridgeAddresses(i);
            _installDepositEthToDrip({
                _drippie: Drippie(mustGetAddress("FaucetDrippie")),
                _name: _makeFaucetDripName(l1BridgeAddress, cfg.dripVersion()),
                _bridge: l1BridgeAddress,
                _target: mustGetAddress("FaucetProxy"),
                _value: cfg.largeOpChainFaucetDripValue(),
                _interval: cfg.largeOpChainFaucetDripInterval()
            });
        }
    }

    /// @notice Installs drips that send funds to the admin wallets for small OP chain faucets
    ///         on the scheduled interval.
    function installSmallOpChainAdminWalletDrips() public {
        require(
            cfg.faucetOnchainAuthModuleAdmin() == cfg.faucetOffchainAuthModuleAdmin(),
            "installSmallOpChainAdminWalletDrips: Only handles identical admin wallet addresses"
        );

        for (uint256 i = 0; i < cfg.getSmallFaucetsL1BridgeAddressesCount(); i++) {
            address l1BridgeAddress = cfg.smallFaucetsL1BridgeAddresses(i);
            _installDepositEthToDrip({
                _drippie: Drippie(mustGetAddress("FaucetDrippie")),
                _name: _makeAdminWalletDripName(l1BridgeAddress, cfg.dripVersion()),
                _bridge: l1BridgeAddress,
                _target: cfg.faucetOnchainAuthModuleAdmin(),
                _value: cfg.opChainAdminWalletDripValue(),
                _interval: cfg.opChainAdminWalletDripInterval()
            });
        }
    }

    /// @notice Installs drips that send funds to the admin wallets for large OP chain faucets
    ///         on the scheduled interval.
    function installLargeOpChainAdminWalletDrips() public {
        require(
            cfg.faucetOnchainAuthModuleAdmin() == cfg.faucetOffchainAuthModuleAdmin(),
            "installLargeOpChainAdminWalletDrips: Only handles identical admin wallet addresses"
        );

        for (uint256 i = 0; i < cfg.getLargeFaucetsL1BridgeAddressesCount(); i++) {
            address l1BridgeAddress = cfg.largeFaucetsL1BridgeAddresses(i);
            _installDepositEthToDrip({
                _drippie: Drippie(mustGetAddress("FaucetDrippie")),
                _name: _makeAdminWalletDripName(l1BridgeAddress, cfg.dripVersion()),
                _bridge: l1BridgeAddress,
                _target: cfg.faucetOnchainAuthModuleAdmin(),
                _value: cfg.opChainAdminWalletDripValue(),
                _interval: cfg.opChainAdminWalletDripInterval()
            });
        }
    }

    /// @notice Installs the FaucetDripV1 drip on the faucet drippie contract.
    function installFaucetDripV1() public broadcast {
        _installBalanceLowDrip({
            _drippie: Drippie(mustGetAddress("FaucetDrippie")),
            _name: "FaucetDripV1",
            _target: mustGetAddress("FaucetProxy"),
            _value: cfg.faucetDripV1Value(),
            _interval: cfg.faucetDripV1Interval(),
            _threshold: cfg.faucetDripV1Threshold()
        });
    }

    /// @notice Installs the FaucetDripV2 drip on the faucet drippie contract.
    function installFaucetDripV2() public broadcast {
        _installBalanceLowDrip({
            _drippie: Drippie(mustGetAddress("FaucetDrippie")),
            _name: "FaucetDripV2",
            _target: mustGetAddress("FaucetProxy"),
            _value: cfg.faucetDripV2Value(),
            _interval: cfg.faucetDripV2Interval(),
            _threshold: cfg.faucetDripV2Threshold()
        });
    }

    /// @notice Installs the FaucetAdminDripV1 drip on the faucet drippie contract.
    function installFaucetAdminDripV1() public broadcast {
        _installBalanceLowDrip({
            _drippie: Drippie(mustGetAddress("FaucetDrippie")),
            _name: "FaucetAdminDripV1",
            _target: mustGetAddress("FaucetProxy"),
            _value: cfg.faucetAdminDripV1Value(),
            _interval: cfg.faucetAdminDripV1Interval(),
            _threshold: cfg.faucetAdminDripV1Threshold()
        });
    }

    /// @notice Installs the GelatoBalanceV2 drip on the faucet drippie contract.
    function installFaucetGelatoBalanceV2() public broadcast {
        Drippie.DripAction[] memory actions = new Drippie.DripAction[](1);
        actions[0] = Drippie.DripAction({
            target: payable(cfg.faucetGelatoTreasury()),
            data: abi.encodeWithSignature(
                "depositFunds(address,address,uint256)",
                cfg.faucetGelatoRecipient(),
                // Gelato represents ETH as 0xeeeee....eeeee
                0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE,
                cfg.faucetGelatoBalanceV1Value()
            ),
            value: cfg.faucetGelatoBalanceV1Value()
        });
        _installDrip({
            _drippie: Drippie(mustGetAddress("FaucetDrippie")),
            _name: "GelatoBalanceV2",
            _config: Drippie.DripConfig({
                reentrant: false,
                interval: cfg.faucetGelatoBalanceV1DripInterval(),
                dripcheck: CheckGelatoLow(mustGetAddress("CheckGelatoLow")),
                checkparams: abi.encode(
                    CheckGelatoLow.Params({
                        recipient: cfg.faucetGelatoRecipient(),
                        threshold: cfg.faucetGelatoThreshold(),
                        treasury: cfg.faucetGelatoTreasury()
                    })
                ),
                actions: actions
            })
        });
    }

    /// @notice Archives the previous OP Chain drip configs.
    function archivePreviousOpChainFaucetsDrippieConfigs() public {
        uint256 drippieOwnerPrivateKey = Config.drippieOwnerPrivateKey();
        vm.startBroadcast(drippieOwnerPrivateKey);

        Drippie drippie = Drippie(mustGetAddress("FaucetDrippie"));
        console.log("Archiving OP Chain faucet drips at %s", address(drippie));
        archivePreviousSmallOpChainFaucetsDrips();
        archivePreviousLargeOpChainFaucetsDrips();

        vm.stopBroadcast();

        console.log("OP chain faucet drip configs successfully installed");
    }

    function archivePreviousSmallOpChainFaucetsDrips() public {
        Drippie drippie = Drippie(mustGetAddress("FaucetDrippie"));
        uint256 arrayLength = cfg.getSmallFaucetsL1BridgeAddressesCount();
        for (uint256 i = 0; i < arrayLength; i++) {
            address l1BridgeAddress = cfg.smallFaucetsL1BridgeAddresses(i);
            drippie.status(_makeFaucetDripName(l1BridgeAddress, cfg.previousDripVersion()), Drippie.DripStatus.PAUSED);
            drippie.status(
                _makeAdminWalletDripName(l1BridgeAddress, cfg.previousDripVersion()), Drippie.DripStatus.PAUSED
            );
            drippie.status(_makeFaucetDripName(l1BridgeAddress, cfg.previousDripVersion()), Drippie.DripStatus.ARCHIVED);
            drippie.status(
                _makeAdminWalletDripName(l1BridgeAddress, cfg.previousDripVersion()), Drippie.DripStatus.ARCHIVED
            );
        }
    }

    function archivePreviousLargeOpChainFaucetsDrips() public {
        Drippie drippie = Drippie(mustGetAddress("FaucetDrippie"));
        uint256 arrayLength = cfg.getLargeFaucetsL1BridgeAddressesCount();
        for (uint256 i = 0; i < arrayLength; i++) {
            address l1BridgeAddress = cfg.largeFaucetsL1BridgeAddresses(i);
            drippie.status(_makeFaucetDripName(l1BridgeAddress, cfg.previousDripVersion()), Drippie.DripStatus.PAUSED);
            drippie.status(
                _makeAdminWalletDripName(l1BridgeAddress, cfg.previousDripVersion()), Drippie.DripStatus.PAUSED
            );
            drippie.status(_makeFaucetDripName(l1BridgeAddress, cfg.previousDripVersion()), Drippie.DripStatus.ARCHIVED);
            drippie.status(
                _makeAdminWalletDripName(l1BridgeAddress, cfg.previousDripVersion()), Drippie.DripStatus.ARCHIVED
            );
        }
    }

    /// @notice Installs the OnChain AuthModule on the Faucet contract.
    function installOnChainAuthModule() public broadcast {
        _installAuthModule({
            _faucet: Faucet(mustGetAddress("FaucetProxy")),
            _name: "OnChainAuthModule",
            _config: Faucet.ModuleConfig({
                name: "OnChainAuthModule",
                enabled: true,
                ttl: cfg.faucetOnchainAuthModuleTtl(),
                amount: cfg.faucetOnchainAuthModuleAmount()
            })
        });
    }

    /// @notice Installs the OffChain AuthModule on the Faucet contract.
    function installOffChainAuthModule() public broadcast {
        _installAuthModule({
            _faucet: Faucet(mustGetAddress("FaucetProxy")),
            _name: "OffChainAuthModule",
            _config: Faucet.ModuleConfig({
                name: "OffChainAuthModule",
                enabled: true,
                ttl: cfg.faucetOffchainAuthModuleTtl(),
                amount: cfg.faucetOffchainAuthModuleAmount()
            })
        });
    }

    /// @notice Installs all of the auth modules in the faucet contract.
    function installFaucetAuthModulesConfigs() public {
        Faucet faucet = Faucet(mustGetAddress("FaucetProxy"));
        console.log("Installing auth modules at %s", address(faucet));
        installOnChainAuthModule();
        installOffChainAuthModule();
        console.log("Faucet Auth Module configs successfully installed");
    }

    /// @notice Generates a drip name for a chain/faucet drip.
    /// @param _l1Bridge The address of the L1 bridge.
    /// @param _version The version of the drip.
    function _makeFaucetDripName(address _l1Bridge, uint256 _version) internal pure returns (string memory) {
        string memory dripNamePrefixWithBridgeAddress = string.concat("faucet-drip-", vm.toString(_l1Bridge));
        string memory versionSuffix = string.concat("-", vm.toString(_version));
        return string.concat(dripNamePrefixWithBridgeAddress, versionSuffix);
    }

    /// @notice Generates a drip name for a chain/admin wallet drip.
    /// @param _l1Bridge The address of the L1 bridge.
    /// @param _version The version of the drip.
    function _makeAdminWalletDripName(address _l1Bridge, uint256 _version) internal pure returns (string memory) {
        string memory dripNamePrefixWithBridgeAddress = string.concat("faucet-admin-drip-", vm.toString(_l1Bridge));
        string memory versionSuffix = string.concat("-", vm.toString(_version));
        return string.concat(dripNamePrefixWithBridgeAddress, versionSuffix);
    }

    // @notice Deploys a contract using the CREATE2 opcode.
    // @param _name The name of the contract.
    // @param _creationCode The contract creation code.
    // @param _constructorParams The constructor parameters.
    function _deployCreate2(
        string memory _name,
        bytes memory _creationCode,
        bytes memory _constructorParams
    )
        internal
        returns (address addr_)
    {
        bytes32 salt = keccak256(bytes(_name));
        bytes memory initCode = abi.encodePacked(_creationCode, _constructorParams);
        address preComputedAddress = computeCreate2Address(salt, keccak256(initCode));
        if (preComputedAddress.code.length > 0) {
            console.log("%s already deployed at %s", _name, preComputedAddress);
            save(_name, preComputedAddress);
            addr_ = preComputedAddress;
        } else {
            assembly {
                addr_ := create2(0, add(initCode, 0x20), mload(initCode), salt)
            }
            require(addr_ != address(0), "deployment failed");
            save(_name, addr_);
            console.log("%s deployed at %s", _name, addr_);
        }
    }

    /// @notice Installs an auth module in the faucet.
    /// @param _faucet The faucet contract.
    /// @param _name The name of the auth module.
    /// @param _config The configuration of the auth module.
    function _installAuthModule(Faucet _faucet, string memory _name, Faucet.ModuleConfig memory _config) internal {
        AdminFaucetAuthModule module = AdminFaucetAuthModule(mustGetAddress(_name));
        if (_faucet.isModuleEnabled(module)) {
            console.log("%s already installed.", _name);
        } else {
            console.log("Installing %s", _name);
            _faucet.configure(module, _config);
            console.log("%s installed successfully", _name);
        }
    }

    /// @notice Installs a drip in the drippie contract.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip.
    /// @param _config The configuration of the drip.
    function _installDrip(Drippie _drippie, string memory _name, Drippie.DripConfig memory _config) internal {
        if (_drippie.getDripStatus(_name) == Drippie.DripStatus.NONE) {
            console.log("installing %s", _name);
            _drippie.create(_name, _config);
            console.log("%s installed successfully", _name);
        } else {
            console.log("%s already installed", _name);
        }

        // Attempt to activate the drip.
        _drippie.status(_name, Drippie.DripStatus.ACTIVE);
    }

    /// @notice Installs a drip that sends ETH to an address if the balance is below a threshold.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip.
    /// @param _target The target address.
    /// @param _value The amount of ETH to send.
    /// @param _interval The interval that must elapse between drips.
    /// @param _threshold The balance threshold.
    function _installBalanceLowDrip(
        Drippie _drippie,
        string memory _name,
        address payable _target,
        uint256 _value,
        uint256 _interval,
        uint256 _threshold
    )
        internal
    {
        Drippie.DripAction[] memory actions = new Drippie.DripAction[](1);
        actions[0] = Drippie.DripAction({ target: _target, data: "", value: _value });
        _installDrip({
            _drippie: _drippie,
            _name: _name,
            _config: Drippie.DripConfig({
                reentrant: false,
                interval: _interval,
                dripcheck: CheckBalanceLow(mustGetAddress("CheckBalanceLow")),
                checkparams: abi.encode(CheckBalanceLow.Params({ target: _target, threshold: _threshold })),
                actions: actions
            })
        });
    }

    /// @notice Installs a drip that sends ETH through the L1StandardBridge on an interval.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip.
    /// @param _bridge The address of the bridge.
    /// @param _target The target address.
    /// @param _value The amount of ETH to send.
    function _installDepositEthToDrip(
        Drippie _drippie,
        string memory _name,
        address _bridge,
        address _target,
        uint256 _value,
        uint256 _interval
    )
        internal
    {
        Drippie.DripAction[] memory actions = new Drippie.DripAction[](1);
        actions[0] = Drippie.DripAction({
            target: payable(_bridge),
            data: abi.encodeCall(L1StandardBridge.depositETHTo, (_target, 200000, "")),
            value: _value
        });
        _installDrip({
            _drippie: _drippie,
            _name: _name,
            _config: Drippie.DripConfig({
                reentrant: false,
                interval: _interval,
                dripcheck: CheckTrue(mustGetAddress("CheckTrue")),
                checkparams: abi.encode(""),
                actions: actions
            })
        });
    }
}

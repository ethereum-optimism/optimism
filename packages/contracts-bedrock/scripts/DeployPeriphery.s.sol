// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { console2 as console } from "forge-std/console2.sol";
import { Script } from "forge-std/Script.sol";

import { IAutomate as IGelato } from "gelato/interfaces/IAutomate.sol";
import { LibDataTypes as GelatoDataTypes } from "gelato/libraries/LibDataTypes.sol";
import { LibTaskId as GelatoTaskId } from "gelato/libraries/LibTaskId.sol";
import { GelatoBytes } from "gelato/vendor/gelato/GelatoBytes.sol";

import { Config } from "scripts/Config.sol";
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
import { CheckSecrets } from "src/periphery/drippie/dripchecks/CheckSecrets.sol";
import { AdminFaucetAuthModule } from "src/periphery/faucet/authmodules/AdminFaucetAuthModule.sol";

import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

/// @title DeployPeriphery
/// @notice Script used to deploy periphery contracts.
contract DeployPeriphery is Script, Artifacts {
    /// @notice Error emitted when an address mismatch is detected.
    error AddressMismatch(string, address, address);

    /// @notice Struct that contains the data for a Gelato task.
    struct GelatoTaskData {
        address taskCreator;
        address execAddress;
        bytes execData;
        GelatoDataTypes.ModuleData moduleData;
        address feeToken;
    }

    /// @notice Deployment configuration.
    PeripheryDeployConfig cfg;

    /// @notice Sets up the deployment script.
    function setUp() public override {
        Artifacts.setUp();
        cfg = new PeripheryDeployConfig(Config.deployConfigPath());
        console.log("Config path: %s", Config.deployConfigPath());
    }

    /// @notice Deploy all of the periphery contracts.
    function run() public {
        console.log("Deploying periphery contracts");

        // Optionally deploy the base dripcheck contracts.
        if (cfg.deployDripchecks()) {
            deployCheckTrue();
            deployCheckBalanceLow();
            deployCheckGelatoLow();
            deployCheckSecrets();
        }

        // Optionally deploy the faucet contracts.
        if (cfg.deployFaucetContracts()) {
            // Deploy faucet contracts.
            deployProxyAdmin();
            deployFaucetProxy();
            deployFaucet();
            deployFaucetDrippie();
            deployOnChainAuthModule();
            deployOffChainAuthModule();

            // Initialize the faucet.
            initializeFaucet();
            installFaucetAuthModulesConfigs();

            // Optionally install OP Chain drip configs.
            if (cfg.installOpChainFaucetsDrips()) {
                installOpChainFaucetsDrippieConfigs();
            }

            // Optionally archive old drip configs.
            if (cfg.archivePreviousOpChainFaucetsDrips()) {
                archivePreviousOpChainFaucetsDrippieConfigs();
            }
        }

        // Optionally deploy the operations contracts.
        if (cfg.deployOperationsContracts()) {
            deployOperationsDrippie();
        }
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

    /// @notice Deploy the Drippie contract for standard operations.
    function deployOperationsDrippie() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "OperationsDrippie",
            _creationCode: type(Drippie).creationCode,
            _constructorParams: abi.encode(cfg.operationsDrippieOwner())
        });

        Drippie drippie = Drippie(payable(addr_));
        require(drippie.owner() == cfg.operationsDrippieOwner());
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

    /// @notice Deploy CheckSecrets contract.
    function deployCheckSecrets() public broadcast returns (address addr_) {
        addr_ = _deployCreate2({
            _name: "CheckSecrets",
            _creationCode: type(CheckSecrets).creationCode,
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

    /// @notice Installs the drip configs in the operations Drippie contract.
    function installOperationsDrippieConfigs() public {
        Drippie drippie = Drippie(mustGetAddress("OperationsDrippie"));
        console.log("Installing operations drips at %s", address(drippie));
        installOperationsSequencerDripV1();
        installOperationsGelatoDripV1();
        installOperationsSecretsDripV1();
        console.log("Operations drip configs successfully installed");
    }

    /// @notice Installs the drip configs in the faucet Drippie contract.
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
                _gelato: IGelato(cfg.gelatoAutomateContract()),
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
                _gelato: IGelato(cfg.gelatoAutomateContract()),
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
                _gelato: IGelato(cfg.gelatoAutomateContract()),
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
                _gelato: IGelato(cfg.gelatoAutomateContract()),
                _drippie: Drippie(mustGetAddress("FaucetDrippie")),
                _name: _makeAdminWalletDripName(l1BridgeAddress, cfg.dripVersion()),
                _bridge: l1BridgeAddress,
                _target: cfg.faucetOnchainAuthModuleAdmin(),
                _value: cfg.opChainAdminWalletDripValue(),
                _interval: cfg.opChainAdminWalletDripInterval()
            });
        }
    }

    /// @notice Installs the OperationsSequencerDripV1 drip on the operations drippie contract.
    function installOperationsSequencerDripV1() public broadcast {
        _installBalanceLowDrip({
            _gelato: IGelato(cfg.gelatoAutomateContract()),
            _drippie: Drippie(mustGetAddress("OperationsDrippie")),
            _name: "OperationsSequencerDripV1",
            _target: cfg.operationsSequencerDripV1Target(),
            _value: cfg.operationsSequencerDripV1Value(),
            _interval: cfg.operationsSequencerDripV1Interval(),
            _threshold: cfg.operationsSequencerDripV1Threshold()
        });
    }

    /// @notice Installs the OperationsGelatoDripV1 drip on the operations drippie contract.
    function installOperationsGelatoDripV1() public broadcast {
        _installGelatoDrip({
            _gelato: IGelato(cfg.gelatoAutomateContract()),
            _drippie: Drippie(mustGetAddress("OperationsDrippie")),
            _name: "OperationsGelatoDripV1",
            _treasury: cfg.gelatoTreasuryContract(),
            _recipient: cfg.operationsGelatoDripV1Recipient(),
            _value: cfg.operationsGelatoDripV1Value(),
            _interval: cfg.operationsGelatoDripV1Interval(),
            _threshold: cfg.operationsGelatoDripV1Threshold()
        });
    }

    /// @notice Installs the OperationsSecretsDripV1 drip on the operations drippie contract.
    function installOperationsSecretsDripV1() public broadcast {
        _installSecretsDrip({
            _gelato: IGelato(cfg.gelatoAutomateContract()),
            _drippie: Drippie(mustGetAddress("OperationsDrippie")),
            _name: "OperationsSecretsDripV1",
            _delay: cfg.operationsSecretsDripV1Delay(),
            _secretHashMustExist: cfg.operationsSecretsDripV1MustExist(),
            _secretHashMustNotExist: cfg.operationsSecretsDripV1MustNotExist(),
            _target: cfg.operationsSecretsDripV1Target(),
            _value: cfg.operationsSecretsDripV1Value(),
            _interval: cfg.operationsSecretsDripV1Interval()
        });
    }

    /// @notice Installs the FaucetDripV1 drip on the faucet drippie contract.
    function installFaucetDripV1() public broadcast {
        _installBalanceLowDrip({
            _gelato: IGelato(cfg.gelatoAutomateContract()),
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
            _gelato: IGelato(cfg.gelatoAutomateContract()),
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
            _gelato: IGelato(cfg.gelatoAutomateContract()),
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
        _installGelatoDrip({
            _gelato: IGelato(cfg.gelatoAutomateContract()),
            _drippie: Drippie(mustGetAddress("FaucetDrippie")),
            _name: "GelatoBalanceV2",
            _treasury: cfg.gelatoTreasuryContract(),
            _recipient: cfg.faucetGelatoRecipient(),
            _value: cfg.faucetGelatoBalanceV1Value(),
            _interval: cfg.faucetGelatoBalanceV1DripInterval(),
            _threshold: cfg.faucetGelatoThreshold()
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

    /// @notice Archives the previous small OP Chain faucet drips.
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

    /// @notice Archives the previous large OP Chain faucet drips.
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

    /// @notice Deploys a contract using the CREATE2 opcode.
    /// @param _name The name of the contract.
    /// @param _creationCode The contract creation code.
    /// @param _constructorParams The constructor parameters.
    function _deployCreate2(
        string memory _name,
        bytes memory _creationCode,
        bytes memory _constructorParams
    )
        internal
        returns (address addr_)
    {
        bytes32 salt = keccak256(abi.encodePacked(bytes(_name), cfg.create2DeploymentSalt()));
        bytes memory initCode = abi.encodePacked(_creationCode, _constructorParams);
        address preComputedAddress = vm.computeCreate2Address(salt, keccak256(initCode));
        if (preComputedAddress.code.length > 0) {
            console.log("%s already deployed at %s", _name, preComputedAddress);
            address savedAddress = getAddress(_name);
            if (savedAddress == address(0)) {
                save(_name, preComputedAddress);
            } else if (savedAddress != preComputedAddress) {
                revert AddressMismatch(_name, preComputedAddress, savedAddress);
            }
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

    /// @notice Generates the data for a Gelato task that would trigger a drip.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip.
    /// @return _taskData Gelato task data.
    function _makeGelatoDripTaskData(
        Drippie _drippie,
        string memory _name
    )
        internal
        view
        returns (GelatoTaskData memory _taskData)
    {
        // Get the drip interval.
        uint256 dripInterval = _drippie.getDripInterval(_name);

        // Set up module types.
        GelatoDataTypes.Module[] memory modules = new GelatoDataTypes.Module[](2);
        modules[0] = GelatoDataTypes.Module.PROXY;
        modules[1] = GelatoDataTypes.Module.TRIGGER;

        // Create arguments for the PROXY and TRIGGER modules.
        bytes[] memory args = new bytes[](2);
        args[0] = abi.encode(_name);
        args[1] = abi.encode(
            GelatoDataTypes.TriggerModuleData({
                triggerType: GelatoDataTypes.TriggerType.TIME,
                triggerConfig: abi.encode(GelatoDataTypes.Time({ nextExec: 0, interval: uint128(dripInterval) }))
            })
        );

        // Create the task data.
        _taskData = GelatoTaskData({
            taskCreator: msg.sender,
            execAddress: address(_drippie),
            execData: abi.encodeCall(Drippie.drip, (_name)),
            moduleData: GelatoDataTypes.ModuleData({ modules: modules, args: args }),
            feeToken: address(0)
        });
    }

    /// @notice Starts a gelato drip task.
    /// @param _gelato The gelato contract.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip being triggered.
    function _startGelatoDripTask(IGelato _gelato, Drippie _drippie, string memory _name) internal {
        GelatoTaskData memory taskData = _makeGelatoDripTaskData({ _drippie: _drippie, _name: _name });
        _gelato.createTask({
            execAddress: taskData.execAddress,
            execData: taskData.execData,
            moduleData: taskData.moduleData,
            feeToken: taskData.feeToken
        });
    }

    /// @notice Pauses a gelato drip task.
    /// @param _gelato The gelato contract.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip being triggered.
    function _pauseGelatoDripTask(IGelato _gelato, Drippie _drippie, string memory _name) internal {
        GelatoTaskData memory taskData = _makeGelatoDripTaskData({ _drippie: _drippie, _name: _name });
        _gelato.cancelTask(
            GelatoTaskId.getTaskId({
                taskCreator: taskData.taskCreator,
                execAddress: taskData.execAddress,
                execSelector: GelatoBytes.memorySliceSelector(taskData.execData),
                moduleData: taskData.moduleData,
                feeToken: taskData.feeToken
            })
        );
    }

    /// @notice Installs a drip in the drippie contract.
    /// @param _gelato The gelato contract.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip.
    /// @param _config The configuration of the drip.
    function _installDrip(
        IGelato _gelato,
        Drippie _drippie,
        string memory _name,
        Drippie.DripConfig memory _config
    )
        internal
    {
        if (_drippie.getDripStatus(_name) == Drippie.DripStatus.NONE) {
            console.log("installing %s", _name);
            _drippie.create(_name, _config);
            _startGelatoDripTask(_gelato, _drippie, _name);
            console.log("%s installed successfully", _name);
        } else {
            console.log("%s already installed", _name);
        }

        // Attempt to activate the drip.
        _drippie.status(_name, Drippie.DripStatus.ACTIVE);
    }

    /// @notice Installs a drip that sends ETH to an address if the balance is below a threshold.
    /// @param _gelato The gelato contract.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip.
    /// @param _target The target address.
    /// @param _value The amount of ETH to send.
    /// @param _interval The interval that must elapse between drips.
    /// @param _threshold The balance threshold.
    function _installBalanceLowDrip(
        IGelato _gelato,
        Drippie _drippie,
        string memory _name,
        address _target,
        uint256 _value,
        uint256 _interval,
        uint256 _threshold
    )
        internal
    {
        Drippie.DripAction[] memory actions = new Drippie.DripAction[](1);
        actions[0] = Drippie.DripAction({ target: payable(_target), data: "", value: _value });
        _installDrip({
            _gelato: _gelato,
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
    /// @param _gelato The gelato contract.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip.
    /// @param _bridge The address of the bridge.
    /// @param _target The target address.
    /// @param _value The amount of ETH to send.
    function _installDepositEthToDrip(
        IGelato _gelato,
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
            _gelato: _gelato,
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

    /// @notice Installs a drip that sends ETH to the Gelato treasury if the balance is below a
    ///         threshold. Balance gets deposited into the account of the recipient.
    /// @param _gelato The gelato contract.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip.
    /// @param _treasury The address of the Gelato treasury.
    /// @param _recipient The address of the recipient.
    /// @param _value The amount of ETH to send.
    /// @param _interval The interval that must elapse between drips.
    function _installGelatoDrip(
        IGelato _gelato,
        Drippie _drippie,
        string memory _name,
        address _treasury,
        address _recipient,
        uint256 _value,
        uint256 _interval,
        uint256 _threshold
    )
        internal
    {
        Drippie.DripAction[] memory actions = new Drippie.DripAction[](1);
        actions[0] = Drippie.DripAction({
            target: payable(_treasury),
            data: abi.encodeWithSignature("depositNative(address)", _recipient),
            value: _value
        });
        _installDrip({
            _gelato: _gelato,
            _drippie: _drippie,
            _name: _name,
            _config: Drippie.DripConfig({
                reentrant: false,
                interval: _interval,
                dripcheck: CheckGelatoLow(mustGetAddress("CheckGelatoLow")),
                checkparams: abi.encode(
                    CheckGelatoLow.Params({ recipient: _recipient, threshold: _threshold, treasury: _treasury })
                ),
                actions: actions
            })
        });
    }

    /// @notice Installs a drip that sends ETH to an account if one given secret is revealed and
    ///         another is not. Drip will stop if the second secret is revealed.
    /// @param _gelato The gelato contract.
    /// @param _drippie The drippie contract.
    /// @param _name The name of the drip.
    /// @param _delay The delay before the drip starts after the first secret is revealed.
    /// @param _secretHashMustExist The hash of the secret that must exist.
    /// @param _secretHashMustNotExist The hash of the secret that must not exist.
    /// @param _target The target address.
    /// @param _value The amount of ETH to send.
    /// @param _interval The interval that must elapse between drips.
    function _installSecretsDrip(
        IGelato _gelato,
        Drippie _drippie,
        string memory _name,
        uint256 _delay,
        bytes32 _secretHashMustExist,
        bytes32 _secretHashMustNotExist,
        address _target,
        uint256 _value,
        uint256 _interval
    )
        internal
    {
        Drippie.DripAction[] memory actions = new Drippie.DripAction[](1);
        actions[0] = Drippie.DripAction({ target: payable(_target), data: "", value: _value });
        _installDrip({
            _gelato: _gelato,
            _drippie: _drippie,
            _name: _name,
            _config: Drippie.DripConfig({
                reentrant: false,
                interval: _interval,
                dripcheck: CheckSecrets(mustGetAddress("CheckSecrets")),
                checkparams: abi.encode(
                    CheckSecrets.Params({
                        delay: _delay,
                        secretHashMustExist: _secretHashMustExist,
                        secretHashMustNotExist: _secretHashMustNotExist
                    })
                ),
                actions: actions
            })
        });
    }
}

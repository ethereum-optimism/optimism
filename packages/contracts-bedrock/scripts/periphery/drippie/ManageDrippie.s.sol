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
import { DrippieConfig } from "scripts/periphery/drippie/DrippieConfig.s.sol";

import { Drippie } from "src/periphery/drippie/Drippie.sol";
import { IDripCheck } from "src/periphery/drippie/IDripCheck.sol";

/// @title ManageDrippie
/// @notice Script for managing drips in the Drippie contract.
contract ManageDrippie is Script, Artifacts {
    /// @notice Struct that contains the data for a Gelato task.
    struct GelatoTaskData {
        address taskCreator;
        address execAddress;
        bytes execData;
        GelatoDataTypes.ModuleData moduleData;
        address feeToken;
    }

    /// @notice Drippie configuration.
    DrippieConfig public cfg;

    /// @notice Modifier that wraps a function in broadcasting.
    modifier broadcast() {
        vm.startBroadcast(msg.sender);
        _;
        vm.stopBroadcast();
    }

    /// @notice Sets up the deployment script.
    function setUp() public override {
        Artifacts.setUp();
        cfg = new DrippieConfig(Config.deployConfigPath());
        console.log("Config path: %s", Config.deployConfigPath());
    }

    /// @notice Runs the management script.
    function run() public {
        console.log("ManageDrippie: running");
        installDrips();
    }

    /// @notice Installs drips in the drippie contract.
    function installDrips() public broadcast {
        console.log("ManageDrippie: installing Drippie config");
        for (uint256 i = 0; i < cfg.dripsLength(); i++) {
            DrippieConfig.FullDripConfig memory drip = abi.decode(cfg.drip(i), (DrippieConfig.FullDripConfig));
            Drippie.DripAction[] memory actions = new Drippie.DripAction[](1);
            actions[0] = Drippie.DripAction({ target: payable(drip.recipient), data: drip.data, value: drip.value });
            _installDrip({
                _gelato: cfg.gelato(),
                _drippie: cfg.drippie(),
                _name: drip.name,
                _config: Drippie.DripConfig({
                    reentrant: false,
                    interval: drip.interval,
                    dripcheck: IDripCheck(mustGetAddress(drip.dripcheck)),
                    checkparams: drip.checkparams,
                    actions: actions
                })
            });
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

        // Grab the status again now that we've installed the drip.
        Drippie.DripStatus status = _drippie.getDripStatus(_name);
        if (status == Drippie.DripStatus.PAUSED) {
            console.log("activating %s", _name);
            _drippie.status(_name, Drippie.DripStatus.ACTIVE);
            console.log("%s activated successfully", _name);
        } else if (status == Drippie.DripStatus.ACTIVE) {
            console.log("%s already active", _name);
        } else {
            // TODO: Better way to handle this?
            console.log("WARNING: % could not be activated", _name);
        }
    }
}

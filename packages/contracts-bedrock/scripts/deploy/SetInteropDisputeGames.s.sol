// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { BaseDeployIO } from "scripts/deploy/BaseDeployIO.sol";
import { IOPContractsManagerMigrator } from "interfaces/L1/opcm/IOPContractsManagerMigrator.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { DummyCaller } from "scripts/libraries/DummyCaller.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";

/// @notice Input for swapping the shared dispute games of an already-interop set to a new super
///         game (e.g. ZKDisputeGame) via OPContractsManagerV2.setInteropDisputeGames.
contract SetInteropDisputeGamesInput is BaseDeployIO {
    address internal _prank;
    address internal _opcm;
    /// @notice The swap input, stored as opaque bytes (an encoded MigrateInput).
    bytes internal _migrateInput;

    function set(bytes4 _sel, address _value) public {
        require(address(_value) != address(0), "SetInteropDisputeGamesInput: cannot set zero address");

        if (_sel == this.prank.selector) _prank = _value;
        else if (_sel == this.opcm.selector) _opcm = _value;
        else revert("SetInteropDisputeGamesInput: unknown selector");
    }

    /// @notice Sets the swap input using the IOPContractsManagerMigrator.MigrateInput type.
    /// @param _sel The selector of the field to set.
    /// @param _value The value to set.
    function set(bytes4 _sel, IOPContractsManagerMigrator.MigrateInput memory _value) public {
        if (_sel == this.migrateInput.selector) _migrateInput = abi.encode(_value);
        else revert("SetInteropDisputeGamesInput: unknown selector");
    }

    function prank() public view returns (address) {
        require(address(_prank) != address(0), "SetInteropDisputeGamesInput: prank not set");
        return _prank;
    }

    function opcm() public view returns (address) {
        require(address(_opcm) != address(0), "SetInteropDisputeGamesInput: opcm not set");
        return _opcm;
    }

    function migrateInput() public view returns (bytes memory) {
        require(_migrateInput.length > 0, "SetInteropDisputeGamesInput: migrateInput not set");
        return _migrateInput;
    }
}

/// @notice Output of the dispute game swap.
contract SetInteropDisputeGamesOutput is BaseDeployIO {
    IDisputeGameFactory internal _disputeGameFactory;

    function set(bytes4 _sel, IDisputeGameFactory _value) public {
        if (_sel == this.disputeGameFactory.selector) _disputeGameFactory = _value;
        else revert("SetInteropDisputeGamesOutput: unknown selector");
    }

    function disputeGameFactory() public view returns (IDisputeGameFactory) {
        require(address(_disputeGameFactory) != address(0), "SetInteropDisputeGamesOutput: not set");
        DeployUtils.assertValidContractAddress(address(_disputeGameFactory));
        return _disputeGameFactory;
    }
}

contract SetInteropDisputeGames is Script {
    function run(SetInteropDisputeGamesInput _udi, SetInteropDisputeGamesOutput _udo) public {
        address opcm = _udi.opcm();
        require(opcm.code.length > 0, "SetInteropDisputeGames: OPCM address has no code");
        require(
            SemverComp.gte(ISemver(opcm).version(), "7.2.1"),
            "SetInteropDisputeGames: OPCM must be v7.2.1 or later (OPCMv2)."
        );

        // Etch DummyCaller contract to mimic the governance multisig that delegatecalls the OPCM.
        address prank = _udi.prank();
        bytes memory code = type(DummyCaller).runtimeCode;
        vm.etch(prank, code);
        vm.store(prank, bytes32(0), bytes32(uint256(uint160(opcm))));
        vm.label(prank, "DummyCaller");

        // Call into the DummyCaller, which performs the delegatecall to the OPCM under the hood.
        // The DummyCaller's fallback reverts on failure, so no need to check success.
        vm.startBroadcast(msg.sender);
        IOPContractsManagerMigrator(prank).setInteropDisputeGames(
            abi.decode(_udi.migrateInput(), (IOPContractsManagerMigrator.MigrateInput))
        );
        vm.stopBroadcast();

        // The shared DisputeGameFactory is unchanged; resolve it for the output.
        _setDisputeGameFactory(_udi, _udo);

        checkOutput(_udi, _udo);
    }

    /// @notice Helper function to set the dispute game factory in the output.
    function _setDisputeGameFactory(SetInteropDisputeGamesInput _udi, SetInteropDisputeGamesOutput _udo) internal {
        IOPContractsManagerMigrator.MigrateInput memory migrateInput =
            abi.decode(_udi.migrateInput(), (IOPContractsManagerMigrator.MigrateInput));
        IOptimismPortal portal = IOptimismPortal(payable(migrateInput.chainSystemConfigs[0].optimismPortal()));
        _udo.set(_udo.disputeGameFactory.selector, portal.disputeGameFactory());
    }

    function checkOutput(SetInteropDisputeGamesInput _udi, SetInteropDisputeGamesOutput _udo) public view {
        IOPContractsManagerMigrator.MigrateInput memory migrateInput =
            abi.decode(_udi.migrateInput(), (IOPContractsManagerMigrator.MigrateInput));

        // All chains still share the same DisputeGameFactory.
        for (uint256 i = 0; i < migrateInput.chainSystemConfigs.length; i++) {
            IOptimismPortal portal = IOptimismPortal(payable(migrateInput.chainSystemConfigs[i].optimismPortal()));
            require(
                IDisputeGameFactory(portal.disputeGameFactory()) == _udo.disputeGameFactory(),
                "SetInteropDisputeGames: disputeGameFactory mismatch"
            );
        }

        // The shared AnchorStateRegistry's respected game type was updated to the new super game.
        IAnchorStateRegistry asr =
            IOptimismPortal(payable(migrateInput.chainSystemConfigs[0].optimismPortal())).anchorStateRegistry();
        require(
            asr.respectedGameType().raw() == migrateInput.startingRespectedGameType.raw(),
            "SetInteropDisputeGames: respected game type not updated"
        );

        // Each game config was actually applied on the shared DisputeGameFactory: enabled games are
        // registered (impl set), disabled games are cleared (impl zeroed). This proves the swap did
        // real work rather than merely flipping the respected type.
        IDisputeGameFactory dgf = _udo.disputeGameFactory();
        for (uint256 i = 0; i < migrateInput.disputeGameConfigs.length; i++) {
            address impl = address(dgf.gameImpls(migrateInput.disputeGameConfigs[i].gameType));
            if (migrateInput.disputeGameConfigs[i].enabled) {
                require(impl != address(0), "SetInteropDisputeGames: enabled game not registered");
            } else {
                require(impl == address(0), "SetInteropDisputeGames: disabled game not cleared");
            }
        }

        // The new respected game type must have a registered implementation.
        require(
            address(dgf.gameImpls(migrateInput.startingRespectedGameType)) != address(0),
            "SetInteropDisputeGames: respected game has no implementation"
        );
    }
}

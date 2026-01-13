// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { BaseDeployIO } from "scripts/deploy/BaseDeployIO.sol";
import { IOPContractsManagerInteropMigrator, IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerMigrator } from "interfaces/L1/opcm/IOPContractsManagerMigrator.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { Duration, Proposal, Hash, GameType } from "src/dispute/lib/Types.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

contract InteropMigrationInput is BaseDeployIO {
    address internal _prank;
    address internal _opcm;
    /// @notice The migrate input is stored as opaque bytes to allow storing both OPCM v1 and v2 migrate inputs.
    bytes internal _migrateInput;

    function set(bytes4 _sel, address _value) public {
        require(address(_value) != address(0), "InteropMigrationInput: cannot set zero address");

        if (_sel == this.prank.selector) _prank = _value;
        else if (_sel == this.opcm.selector) _opcm = _value;
        else revert("InteropMigrationInput: unknown selector");
    }

    /// @notice Sets the migrate input using the IOPContractsManagerInteropMigrator.MigrateInput type,
    ///         this is used when migrating chains using OPCM v1.
    /// @param _sel The selector of the field to set.
    /// @param _value The value to set.
    function set(bytes4 _sel, IOPContractsManagerInteropMigrator.MigrateInput memory _value) public {
        if (_sel == this.migrateInput.selector) _migrateInput = abi.encode(_value);
        else revert("InteropMigrationInput: unknown selector");
    }

    /// @notice Sets the migrate input using the IOPContractsManagerMigrator.MigrateInput type,
    ///         this is used when migrating chains using OPCM v2.
    /// @param _sel The selector of the field to set.
    /// @param _value The value to set.
    function set(bytes4 _sel, IOPContractsManagerMigrator.MigrateInput memory _value) public {
        if (_sel == this.migrateInput.selector) _migrateInput = abi.encode(_value);
        else revert("InteropMigrationInput: unknown selector");
    }

    function prank() public view returns (address) {
        require(address(_prank) != address(0), "InteropMigrationInput: prank not set");
        return _prank;
    }

    function opcm() public view returns (address) {
        require(address(_opcm) != address(0), "InteropMigrationInput: opcm not set");
        return _opcm;
    }

    function migrateInput() public view returns (bytes memory) {
        require(_migrateInput.length > 0, "InteropMigrationInput: migrateInput not set");
        return _migrateInput;
    }
}

contract InteropMigrationOutput is BaseDeployIO {
    IDisputeGameFactory internal _disputeGameFactory;

    function set(bytes4 _sel, IDisputeGameFactory _value) public {
        if (_sel == this.disputeGameFactory.selector) _disputeGameFactory = _value;
        else revert("InteropMigrationOutput: unknown selector");
    }

    function disputeGameFactory() public view returns (IDisputeGameFactory) {
        require(address(_disputeGameFactory) != address(0), "InteropMigrationOutput: not set");
        DeployUtils.assertValidContractAddress(address(_disputeGameFactory));
        return _disputeGameFactory;
    }
}

contract InteropMigration is Script {
    /// @notice Whether to use OPCM v2.
    bool internal _useOPCMv2;

    function run(InteropMigrationInput _imi, InteropMigrationOutput _imo) public {
        // Determine OPCM version by checking the semver or if the OPCM address is set. OPCM v2 starts at version 7.0.0.
        IOPContractsManager opcm = IOPContractsManager(_imi.opcm());
        require(address(opcm).code.length > 0, "InteropMigration: OPCM address has no code");
        _useOPCMv2 = SemverComp.gte(opcm.version(), "7.0.0");

        // Etch DummyCaller contract. This contract is used to mimic the contract that is used
        // as the source of the delegatecall to the OPCM. In practice this will be the governance
        // 2/2 or similar.
        address prank = _imi.prank();
        bytes memory code = _getDummyCallerCode();
        vm.etch(prank, code);
        vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(opcm)))));
        vm.label(prank, "DummyCaller");

        // Call into the DummyCaller. This will perform the delegatecall under the hood and
        // return the result.
        (bool success,) = _migrate(prank, _imi.migrateInput());
        require(success, "InteropMigration: migrate failed");

        // After migration all portals will have the same DGF
        _setDisputeGameFactory(_imi, _imo);

        checkOutput(_imi, _imo);
    }

    /// @notice Helper function to get the proper dummy caller code based on the OPCM version.
    /// @return code The code of the dummy caller.
    function _getDummyCallerCode() internal view returns (bytes memory) {
        if (_useOPCMv2) return vm.getDeployedCode("InteropMigration.s.sol:DummyCallerV2");
        else return vm.getDeployedCode("InteropMigration.s.sol:DummyCallerV1");
    }

    /// @notice Helper function to migrate the OPCM based on the OPCM version. Performs the decoding of the migrate
    /// input and the delegatecall to the OPCM.
    /// @param _prank The address of the dummy caller contract.
    /// @param _migrateInput The migrate input.
    /// @return success Whether the migration succeeded.
    /// @return result The result of the migration (bool, bytes memory).
    function _migrate(address _prank, bytes memory _migrateInput) internal returns (bool, bytes memory) {
        vm.broadcast(msg.sender);
        if (_useOPCMv2) {
            return DummyCallerV2(_prank).migrate(abi.decode(_migrateInput, (IOPContractsManagerMigrator.MigrateInput)));
        } else {
            return DummyCallerV1(_prank).migrate(
                abi.decode(_migrateInput, (IOPContractsManagerInteropMigrator.MigrateInput))
            );
        }
    }

    /// @notice Helper function to set the dispute game factory in the output based on the OPCM version.
    /// @param _imi The migration input.
    /// @param _imo The migration output.
    function _setDisputeGameFactory(InteropMigrationInput _imi, InteropMigrationOutput _imo) internal {
        if (_useOPCMv2) {
            IOPContractsManagerMigrator.MigrateInput memory inputV2 =
                abi.decode(_imi.migrateInput(), (IOPContractsManagerMigrator.MigrateInput));
            IOptimismPortal portal = IOptimismPortal(payable(inputV2.chainSystemConfigs[0].optimismPortal()));
            _imo.set(_imo.disputeGameFactory.selector, portal.disputeGameFactory());
        } else {
            IOPContractsManagerInteropMigrator.MigrateInput memory inputV1 =
                abi.decode(_imi.migrateInput(), (IOPContractsManagerInteropMigrator.MigrateInput));
            IOptimismPortal portal =
                IOptimismPortal(payable(inputV1.opChainConfigs[0].systemConfigProxy.optimismPortal()));
            _imo.set(_imo.disputeGameFactory.selector, portal.disputeGameFactory());
        }
    }

    function checkOutput(InteropMigrationInput _imi, InteropMigrationOutput _imo) public view {
        if (_useOPCMv2) {
            IOPContractsManagerMigrator.MigrateInput memory inputV2 =
                abi.decode(_imi.migrateInput(), (IOPContractsManagerMigrator.MigrateInput));

            for (uint256 i = 0; i < inputV2.chainSystemConfigs.length; i++) {
                IOptimismPortal portal = IOptimismPortal(payable(inputV2.chainSystemConfigs[i].optimismPortal()));
                require(
                    IDisputeGameFactory(portal.disputeGameFactory()) == _imo.disputeGameFactory(),
                    "InteropMigration: disputeGameFactory mismatch"
                );
            }
        } else {
            IOPContractsManagerInteropMigrator.MigrateInput memory inputV1 =
                abi.decode(_imi.migrateInput(), (IOPContractsManagerInteropMigrator.MigrateInput));

            for (uint256 i = 0; i < inputV1.opChainConfigs.length; i++) {
                IOptimismPortal portal =
                    IOptimismPortal(payable(inputV1.opChainConfigs[i].systemConfigProxy.optimismPortal()));
                require(
                    IDisputeGameFactory(portal.disputeGameFactory()) == _imo.disputeGameFactory(),
                    "InteropMigration: disputeGameFactory mismatch"
                );
            }
        }
    }
}

contract DummyCallerV1 {
    address internal _opcmAddr;

    function migrate(IOPContractsManagerInteropMigrator.MigrateInput memory _migrateInput)
        external
        returns (bool, bytes memory)
    {
        bytes memory data = abi.encodeCall(IOPContractsManager.migrate, _migrateInput);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}

contract DummyCallerV2 {
    address internal _opcmAddr;

    function migrate(IOPContractsManagerMigrator.MigrateInput memory _migrateInput)
        external
        returns (bool, bytes memory)
    {
        bytes memory data = abi.encodeCall(IOPContractsManagerV2.migrate, _migrateInput);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}

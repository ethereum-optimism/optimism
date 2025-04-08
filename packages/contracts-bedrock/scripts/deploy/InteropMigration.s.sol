// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { BaseDeployIO } from "scripts/deploy/BaseDeployIO.sol";
import { IOPContractsManagerInteropMigrator, IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { Duration, Proposal, Hash } from "src/dispute/lib/Types.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";

contract InteropMigrationInput is BaseDeployIO {
    address internal _prank;
    IOPContractsManager internal _opcm;

    bool internal _usePermissionlessGame;

    // starting anchor proposal
    bytes32 internal _startingAnchorRoot;
    uint256 internal _startingAnchorL2SequenceNumber;

    // game parameters
    address internal _proposer;
    address internal _challenger;
    uint256 internal _maxGameDepth;
    uint256 internal _splitDepth;
    uint256 internal _initBond;
    uint256 internal _clockExtension;
    uint256 internal _maxClockDuration;

    bytes internal _opChainConfigs;

    function set(bytes4 _sel, address _value) public {
        require(address(_value) != address(0), "InteropMigrationInput: cannot set zero address");

        if (_sel == this.prank.selector) _prank = _value;
        else if (_sel == this.opcm.selector) _opcm = IOPContractsManager(_value);
        else if (_sel == this.proposer.selector) _proposer = _value;
        else if (_sel == this.challenger.selector) _challenger = _value;
        else revert("InteropMigrationInput: unknown selector");
    }

    function set(bytes4 _sel, bool _value) public {
        if (_sel == this.usePermissionlessGame.selector) _usePermissionlessGame = _value;
        else revert("InteropMigrationInput: unknown selector");
    }

    function set(bytes4 _sel, uint256 _value) public {
        if (_sel == this.maxGameDepth.selector) {
            require(_value != 0, "InteropMigrationInput: maxGameDepth cannot be 0");
            _maxGameDepth = _value;
        } else if (_sel == this.splitDepth.selector) {
            require(_value != 0, "InteropMigrationInput: splitDepth cannot be 0");
            _splitDepth = _value;
        } else if (_sel == this.initBond.selector) {
            require(_value != 0, "InteropMigrationInput: initBond cannot be 0");
            _initBond = _value;
        } else if (_sel == this.clockExtension.selector) {
            require(_value <= type(uint64).max, "InteropMigrationInput: clockExtension must fit inside uint64");
            require(_value != 0, "InteropMigrationInput: clockExtension cannot be 0");
            _clockExtension = _value;
        } else if (_sel == this.maxClockDuration.selector) {
            require(_value <= type(uint64).max, "InteropMigrationInput: maxClockDuration must fit inside uint64");
            require(_value != 0, "InteropMigrationInput: maxClockDuration cannot be 0");
            _maxClockDuration = _value;
        } else if (_sel == this.startingAnchorL2SequenceNumber.selector) {
            require(_value != 0, "InteropMigrationInput: startingAnchorL2SequenceNumber cannot be 0");
            _startingAnchorL2SequenceNumber = _value;
        } else {
            revert("InteropMigrationInput: unknown selector");
        }
    }

    function set(bytes4 _sel, bytes32 _value) public {
        if (_sel == this.startingAnchorRoot.selector) {
            require(_value != bytes32(0), "InteropMigrationInput: startingAnchorRoot cannot be 0");
            _startingAnchorRoot = _value;
        } else {
            revert("InteropMigrationInput: unknown selector");
        }
    }

    function set(bytes4 _sel, IOPContractsManager.OpChainConfig[] memory _value) public {
        require(_value.length > 0, "InteropMigrationInput: cannot set empty array");

        if (_sel == this.opChainConfigs.selector) _opChainConfigs = abi.encode(_value);
        else revert("InteropMigrationInput: unknown selector");
    }

    function prank() public view returns (address) {
        require(address(_prank) != address(0), "InteropMigrationInput: prank not set");
        return _prank;
    }

    function opcm() public view returns (IOPContractsManager) {
        require(address(_opcm) != address(0), "InteropMigrationInput: not set");
        return _opcm;
    }

    function usePermissionlessGame() public view returns (bool) {
        return _usePermissionlessGame;
    }

    function proposer() public view returns (address) {
        require(address(_proposer) != address(0), "InteropMigrationInput: proposer not set");
        return _proposer;
    }

    function challenger() public view returns (address) {
        require(address(_challenger) != address(0), "InteropMigrationInput: challenger not set");
        return _challenger;
    }

    function maxGameDepth() public view returns (uint256) {
        require(_maxGameDepth > 0, "InteropMigrationInput: maxGameDepth not set");
        return _maxGameDepth;
    }

    function splitDepth() public view returns (uint256) {
        require(_splitDepth > 0, "InteropMigrationInput: splitDepth not set");
        return _splitDepth;
    }

    function initBond() public view returns (uint256) {
        require(_initBond > 0, "InteropMigrationInput: initBond not set");
        return _initBond;
    }

    function clockExtension() public view returns (uint256) {
        require(_clockExtension > 0, "InteropMigrationInput: clockExtension not set");
        return _clockExtension;
    }

    function maxClockDuration() public view returns (uint256) {
        require(_maxClockDuration > 0, "InteropMigrationInput: maxClockDuration not set");
        return _maxClockDuration;
    }

    function startingAnchorRoot() public view returns (bytes32) {
        require(_startingAnchorRoot != bytes32(0), "InteropMigrationInput: startingAnchorRoot not set");
        return _startingAnchorRoot;
    }

    function startingAnchorL2SequenceNumber() public view returns (uint256) {
        require(_startingAnchorL2SequenceNumber > 0, "InteropMigrationInput: startingAnchorL2SequenceNumber not set");
        return _startingAnchorL2SequenceNumber;
    }

    function opChainConfigs() public view returns (bytes memory) {
        require(_opChainConfigs.length > 0, "InteropMigrationInput: not set");
        return _opChainConfigs;
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
    function run(InteropMigrationInput _imi, InteropMigrationOutput _imo) public {
        IOPContractsManager opcm = _imi.opcm();
        IOPContractsManager.OpChainConfig[] memory opChainConfigs =
            abi.decode(_imi.opChainConfigs(), (IOPContractsManager.OpChainConfig[]));

        IOPContractsManagerInteropMigrator.MigrateInput memory inputs = IOPContractsManagerInteropMigrator.MigrateInput({
            usePermissionlessGame: _imi.usePermissionlessGame(),
            startingAnchorRoot: Proposal({
                root: Hash.wrap(_imi.startingAnchorRoot()),
                l2SequenceNumber: _imi.startingAnchorL2SequenceNumber()
            }),
            gameParameters: IOPContractsManagerInteropMigrator.GameParameters({
                proposer: _imi.proposer(),
                challenger: _imi.challenger(),
                maxGameDepth: _imi.maxGameDepth(),
                splitDepth: _imi.splitDepth(),
                initBond: _imi.initBond(),
                clockExtension: Duration.wrap(uint64(_imi.clockExtension())),
                maxClockDuration: Duration.wrap(uint64(_imi.maxClockDuration()))
            }),
            opChainConfigs: opChainConfigs
        });
        // Etch DummyCaller contract. This contract is used to mimic the contract that is used
        // as the source of the delegatecall to the OPCM. In practice this will be the governance
        // 2/2 or similar.
        address prank = _imi.prank();
        bytes memory code = vm.getDeployedCode("InteropMigration.s.sol:DummyCaller");
        vm.etch(prank, code);
        vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(opcm)))));
        vm.label(prank, "DummyCaller");

        // Call into the DummyCaller. This will perform the delegatecall under the hood and
        // return the result.
        vm.broadcast(msg.sender);
        (bool success,) = DummyCaller(prank).migrate(inputs);
        require(success, "InteropMigration: migrate failed");

        // After migration all portals will have the same DGF
        IOptimismPortal portal = IOptimismPortal(payable(opChainConfigs[0].systemConfigProxy.optimismPortal()));
        _imo.set(_imo.disputeGameFactory.selector, portal.disputeGameFactory());

        checkOutput(_imi, _imo);
    }

    function checkOutput(InteropMigrationInput _imi, InteropMigrationOutput _imo) public view {
        IOPContractsManager.OpChainConfig[] memory opChainConfigs =
            abi.decode(_imi.opChainConfigs(), (IOPContractsManager.OpChainConfig[]));

        for (uint256 i = 0; i < opChainConfigs.length; i++) {
            IOptimismPortal portal = IOptimismPortal(payable(opChainConfigs[i].systemConfigProxy.optimismPortal()));
            require(
                IDisputeGameFactory(portal.disputeGameFactory()) == _imo.disputeGameFactory(),
                "InteropMigration: disputeGameFactory mismatch"
            );
        }
    }
}

contract DummyCaller {
    address internal _opcmAddr;

    function migrate(IOPContractsManagerInteropMigrator.MigrateInput memory _migrateInput)
        external
        returns (bool, bytes memory)
    {
        bytes memory data = abi.encodeCall(DummyCaller.migrate, _migrateInput);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}

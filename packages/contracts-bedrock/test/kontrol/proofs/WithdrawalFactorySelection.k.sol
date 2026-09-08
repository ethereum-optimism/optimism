// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { DeploymentSummaryFaultProofs } from "./utils/DeploymentSummaryFaultProofs.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { GameType, Timestamp } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";

/// @notice Actual factory selection over arbitrary application storage and a fixed deployed implementation.
///         Portal composition and factory creation/history preservation are separate obligations.
contract WithdrawalFactorySelectionKontrol is DeploymentSummaryFaultProofs, KontrolUtils {
    IDisputeGameFactory internal factory;

    function setUp() public {
        factory = IOptimismPortal2(payable(optimismPortalProxyAddress)).anchorStateRegistry().disputeGameFactory();
    }

    /// @notice Every index selects its actual packed word or reverts when outside the array.
    function prove_factorySelection_equivalence(uint256 _index) external {
        bytes32 implementation = vm.load(address(factory), Constants.PROXY_IMPLEMENTATION_ADDRESS);
        kevm.symbolicStorage(address(factory));
        vm.store(address(factory), Constants.PROXY_IMPLEMENTATION_ADDRESS, implementation);
        _checkFactorySelection(_index);
    }

    /// @notice Distinct entries admit a nonzero index and reject the first out-of-range index.
    function prove_factorySelection_witness_succeeds() external {
        kevm.setGas(30_000_000);
        uint256 base = uint256(keccak256(abi.encode(uint256(104))));
        vm.store(address(factory), bytes32(uint256(104)), bytes32(uint256(2)));
        vm.store(address(factory), bytes32(base), bytes32((uint256(1) << 224) | (uint256(11) << 160) | 0x1234));
        unchecked {
            vm.store(address(factory), bytes32(base + 1), bytes32((uint256(2) << 224) | (uint256(22) << 160) | 0x5678));
        }
        assert(_checkFactorySelection(1));
        assert(!_checkFactorySelection(2));
    }

    function _checkFactorySelection(uint256 _index) internal view returns (bool accepted_) {
        uint256 count = uint256(vm.load(address(factory), bytes32(uint256(104))));
        uint256 slot;
        unchecked {
            slot = uint256(keccak256(abi.encode(uint256(104)))) + _index;
        }
        // Read the same state as the getter, including possible modular slot aliases.
        uint256 packed = uint256(vm.load(address(factory), bytes32(slot)));
        bytes memory result;
        (accepted_, result) = address(factory).staticcall(abi.encodeCall(factory.gameAtIndex, (_index)));
        assert(accepted_ == (_index < count));
        if (accepted_) {
            (GameType kind, Timestamp created, IDisputeGame candidate) =
                abi.decode(result, (GameType, Timestamp, IDisputeGame));
            assert(GameType.unwrap(kind) == uint32(packed >> 224));
            assert(Timestamp.unwrap(created) == uint64(packed >> 160));
            assert(address(candidate) == address(uint160(packed)));
        }
    }
}

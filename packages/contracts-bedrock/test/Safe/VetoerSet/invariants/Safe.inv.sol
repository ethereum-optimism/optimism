// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe, Enum, OwnerManager, ModuleManager } from "safe-contracts/Safe.sol";
import { LibSort } from "solady/utils/LibSort.sol";

import { OwnerGuard } from "src/Safe/VetoerSet/OwnerGuard.sol";
import { AddOwnerModule } from "src/Safe/VetoerSet/AddOwnerModule.sol";
import { VetoModule } from "src/Safe/VetoerSet/VetoModule.sol";

import "forge-std/Test.sol";

contract Handler is Test {
    Safe public immutable safe;
    OwnerGuard public immutable ownerGuard;
    AddOwnerModule public immutable addOwnerModule;
    VetoModule public immutable vetoModule;

    address private immutable _opFoundation;
    address private immutable _delayedVetoable;

    mapping(address => Account) private _owners;

    constructor() {
        // Deploy the `Safe` Account.
        // NOTE: Mimic a factory deployment via `getDeployedCode` + `etch`.
        address safeAddr = makeAddr("Safe");
        bytes memory code = vm.getDeployedCode("Safe.sol");
        vm.etch(safeAddr, code);

        safe = Safe(payable(safeAddr));

        // Create the initial owner.
        Account memory owner = _createNewOwner(string.concat("Owner_", vm.toString(address(0x0))));
        address[] memory owners = new address[](1);
        owners[0] = owner.addr;

        // Setup the Safe Account with default settings and only one registered `_owner`.
        // TODO: For some reasons calling this method makes `Safe` a targeted contract in the fuzzing campaign.
        safe.setup({
            _owners: owners,
            _threshold: 1,
            to: address(0x0),
            data: "",
            fallbackHandler: address(0),
            paymentToken: address(0),
            payment: 0,
            paymentReceiver: payable(address(0))
        });

        // Deploy the guards and modules contracts.
        ownerGuard = new OwnerGuard(safe);
        _opFoundation = makeAddr("OPFoundation");
        addOwnerModule = new AddOwnerModule(safe, _opFoundation);
        _delayedVetoable = makeAddr("DelayedVetoable");
        vetoModule = new VetoModule(safe, _delayedVetoable);

        // Setup the guard and modules on the Safe Account.
        // NOTE: Bypass `authorized` access control during setup by pranking the Safe Account.
        vm.startPrank(address(safe));
        safe.setGuard(address(ownerGuard));
        safe.enableModule(address(addOwnerModule));
        safe.enableModule(address(vetoModule));
        vm.stopPrank();
    }

    ////////////////////////////////////////////////////////////////////////////////////////////////////
    //                                      Safe Owner Functions                                      //
    ////////////////////////////////////////////////////////////////////////////////////////////////////

    /// @dev Calls `addOwnerWithThreshold` on the `Safe` contract.
    /// @dev Bound `threshold` to [1, safeOwnerCount + 1].
    /// @dev Make all the current owners sign the transaction.
    function addOwnerWithThreshold(address ownerAddrSeed, uint256 threshold) external {
        threshold = bound(threshold, 1, safe.getOwners().length + 1);
        Account memory owner = _createNewOwner(string.concat("Owner_", vm.toString(ownerAddrSeed)));

        _executeAuthorizedCall(abi.encodeCall(OwnerManager.addOwnerWithThreshold, (owner.addr, threshold)));
    }

    /// @dev Calls `removeOwner` on the `Safe` contract.
    /// @dev Bound `threshold` to [1, safeOwnerCount + 1].
    /// @dev Make all the current owners sign the transaction.
    function removeOwner(uint256 ownerIndex, uint256 threshold) external {
        address[] memory owners = safe.getOwners();
        uint256 ownerCount = owners.length;

        // Can't remove all the owners.
        vm.assume(ownerCount > 1);
        threshold = bound(threshold, 1, ownerCount - 1);

        // Get a random owner to remove from `ownerIndex` and the preceding one.
        ownerIndex = ownerIndex % ownerCount;
        address owner = owners[ownerIndex];
        address prevOwner = ownerIndex == 0 ? address(0x1) : owners[ownerIndex - 1];

        _executeAuthorizedCall(abi.encodeCall(OwnerManager.removeOwner, (prevOwner, owner, threshold)));
    }

    /// @dev Calls `swapOwner` on the `Safe` contract.
    /// @dev Make all the current owners sign the transaction.
    function swapOwner(address ownerAddrSeed, uint256 ownerIndex) external {
        Account memory newOwner = _createNewOwner(string.concat("Owner_", vm.toString(ownerAddrSeed)));

        // Get a random owner to remove from `ownerIndex` and the preceding one.
        address[] memory owners = safe.getOwners();
        ownerIndex = ownerIndex % owners.length;
        address ownerToRemove = owners[ownerIndex];
        address prevOwner = ownerIndex == 0 ? address(0x1) : owners[ownerIndex - 1];

        _executeAuthorizedCall(abi.encodeCall(OwnerManager.swapOwner, (prevOwner, ownerToRemove, newOwner.addr)));
    }

    /// @dev Calls `changeThreshold` on the `Safe` contract.
    /// @dev Bound `threshold` to [1, safeOwnerCount].
    /// @dev Make all the current owners sign the transaction.
    function changeThreshold(uint256 threshold) external {
        threshold = bound(threshold, 1, safe.getOwners().length);
        _executeAuthorizedCall(abi.encodeCall(OwnerManager.changeThreshold, (threshold)));
    }

    ////////////////////////////////////////////////////////////////////////////////////////////////////
    //                                     OwnerGuard functions                                       //
    ////////////////////////////////////////////////////////////////////////////////////////////////////

    /// @dev Calls `updateMaxOwnerCount` on the `OwnerGuard` contract.
    /// @dev Prank the Safe Account to bypass access control check.
    function updateMaxOwnerCount(uint8 newMaxOwnerCount) external {
        vm.prank(address(safe));
        ownerGuard.updateMaxOwnerCount(newMaxOwnerCount);
    }

    ////////////////////////////////////////////////////////////////////////////////////////////////////
    //                               AddOwnerModule Functions                               //
    ////////////////////////////////////////////////////////////////////////////////////////////////////

    /// @dev Calls `addOwner` on the `AddOwnerModule` contract.
    /// @dev Prank `opFoundation` to bypass access control check.
    function addOwner(address ownerAddrSeed) external {
        Account memory owner = _createNewOwner(string.concat("OP_Owner_", vm.toString(ownerAddrSeed)));

        vm.prank(_opFoundation);
        addOwnerModule.addOwner(owner.addr);
    }

    ////////////////////////////////////////////////////////////////////////////////////////////////////
    //                                      VetoModule Functions                                      //
    ////////////////////////////////////////////////////////////////////////////////////////////////////

    /// @dev Calls `veto` on the `VetoModule` contract.
    /// @dev Prank a Safe Account owner randomly based on `ownerIndex` to bypass access control check.
    function veto(uint256 ownerIndex) external {
        address[] memory owners = safe.getOwners();
        address owner = owners[ownerIndex % owners.length];

        vm.prank(owner);
        vetoModule.veto();
    }

    ////////////////////////////////////////////////////////////////////////////////////////////////////
    //                                         Private Helpers                                        //
    ////////////////////////////////////////////////////////////////////////////////////////////////////

    /// @dev Create a new owner based on `ownerAddrSeed` and registers it in the `_owners` mapping.
    function _createNewOwner(string memory seed) private returns (Account memory owner) {
        owner = makeAccount(seed);
        _owners[owner.addr] = owner;
    }

    /// @dev Compute all the current owner signatures for the given `txHash`.
    function _computeSignatures(bytes32 txHash) private view returns (bytes memory signatures) {
        address[] memory ownerAddrs = safe.getOwners();
        LibSort.sort(ownerAddrs);

        for (uint256 i = 0; i < ownerAddrs.length; i++) {
            address ownerAddr = ownerAddrs[i];
            Account memory owner = _owners[ownerAddr];
            (uint8 v, bytes32 r, bytes32 s) = vm.sign(owner.key, txHash);

            signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
        }
    }

    /// @dev Calls `execTransaction` on the Safe Account, using whatever the provided `data` and making sure all the
    ///      current owner signed the payload.
    function _executeAuthorizedCall(bytes memory data) private {
        // Compute the transaction hash.
        bytes32 txHash = safe.getTransactionHash({
            to: address(safe),
            value: 0,
            data: data,
            operation: Enum.Operation.Call,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(address(0)),
            _nonce: safe.nonce()
        });

        // Get all the owner signatures.
        bytes memory signatures = _computeSignatures(txHash);

        // Execute the transaction.
        safe.execTransaction({
            to: address(safe),
            value: 0,
            data: data,
            operation: Enum.Operation.Call,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(address(0)),
            signatures: signatures
        });
    }
}

contract InvariantSafe is Test {
    Handler private _handler;

    function setUp() public {
        _handler = new Handler();

        excludeContract(address(_handler.safe()));
        targetContract(address(_handler));
    }

    function invariant_MaxOwnerCountAlwaysAboveOrEqualToSafeOwnerCount() public {
        uint256 maxOwnerCount = _handler.ownerGuard().maxOwnerCount();
        uint256 safeOwnerCount = _handler.safe().getOwners().length;

        assertGe(maxOwnerCount, safeOwnerCount);
    }

    function invariant_SafeThresholdAlwaysEqualsToOwnerGuardThreshold() public {
        uint256 safeOwnerCount = _handler.safe().getOwners().length;
        uint256 ownerGuardThreshold = _handler.ownerGuard().checkNewOwnerCount(safeOwnerCount);
        uint256 safeThreshold = _handler.safe().getThreshold();

        assertEq(ownerGuardThreshold, safeThreshold);
    }
}

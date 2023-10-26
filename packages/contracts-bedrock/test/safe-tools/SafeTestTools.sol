// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0 <0.9.0;

import "forge-std/Test.sol";
import "scripts/libraries/LibSort.sol";
import { Safe as GnosisSafe, OwnerManager, ModuleManager, GuardManager } from "safe-contracts/Safe.sol";
import { SafeProxyFactory as GnosisSafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { SignMessageLib } from "safe-contracts/libraries/SignMessageLib.sol";
import "./CompatibilityFallbackHandler_1_3_0.sol";

// Tools to simplify testing Safe contracts
// Author: Colin Nielsen (https://github.com/colinnielsen/safe-tools)

address constant VM_ADDR = 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D;
bytes12 constant ADDR_MASK = 0xffffffffffffffffffffffff;

/// @dev The address of the first owner in the linked list of owners
address constant SENTINEL_OWNERS = address(0x1);

/// @dev Get the address from a private key
function getAddr(uint256 pk) pure returns (address) {
    return Vm(VM_ADDR).addr(pk);
}

/// @dev Get arrays of addresses and private keys. The arrays are sorted by address, and the addresses are labelled
function makeAddrsAndKeys(uint256 num) returns (address[] memory addrs, uint256[] memory keys) {
    keys = new uint256[](num);
    addrs = new address[](num);
    for (uint256 i; i < num; i++) {
        uint256 key = uint256(keccak256(abi.encodePacked(i)));
        keys[i] = key;
    }

    for (uint256 i; i < num; i++) {
        addrs[i] = Vm(VM_ADDR).addr(keys[i]);
        Vm(VM_ADDR).label(getAddr(keys[i]), string.concat("SAFETEST: Signer ", string(abi.encodePacked(bytes32(i)))));
    }
}

/// @dev Encode a smart contract wallet as a private key
function encodeSmartContractWalletAsPK(address addr) pure returns (uint256 encodedPK) {
    assembly {
        let addr_b32 := addr
        encodedPK := or(addr, ADDR_MASK)
    }
}

/// @dev Decode a smart contract wallet as an address from a private key
function decodeSmartContractWalletAsAddress(uint256 pk) pure returns (address decodedAddr) {
    assembly {
        let addr := shl(96, pk)
        decodedAddr := shr(96, addr)
    }
}

/// @dev Checks if a private key is an encoded smart contract address
function isSmartContractPK(uint256 pk) pure returns (bool isEncoded) {
    assembly {
        isEncoded := eq(shr(160, pk), shr(160, ADDR_MASK))
    }
}

library Sort {
    /// @dev Sorts an array of addresses in place
    function sort(address[] memory arr) public pure returns (address[] memory) {
        LibSort.sort(arr);
        return arr;
    }
}

/// @dev Sorts an array of private keys by the computed address
///      If the private key is a smart contract wallet, it will be decoded and sorted by the address
function sortPKsByComputedAddress(uint256[] memory _pks) pure returns (uint256[] memory) {
    uint256[] memory sortedPKs = new uint256[](_pks.length);

    address[] memory addresses = new address[](_pks.length);
    bytes32[2][] memory accounts = new bytes32[2][](_pks.length);

    for (uint256 i; i < _pks.length; i++) {
        uint256 pk = _pks[i];
        address signer = getAddr(pk);
        if (isSmartContractPK(pk)) {
            signer = decodeSmartContractWalletAsAddress(pk);
        }
        addresses[i] = signer;
        accounts[i][0] = bytes32(abi.encode(signer));
        accounts[i][1] = bytes32(pk);
    }

    addresses = Sort.sort(addresses);

    uint256 found;
    for (uint256 j; j < addresses.length; j++) {
        address signer = addresses[j];
        uint256 pk;
        for (uint256 k; k < accounts.length; k++) {
            if (address(uint160(uint256(accounts[k][0]))) == signer) {
                pk = uint256(accounts[k][1]);
                found++;
            }
        }

        sortedPKs[j] = pk;
    }

    if (found < _pks.length) {
        revert("SAFETESTTOOLS: issue with private key sorting, please open a ticket on github");
    }
    return sortedPKs;
}

/// @dev A minimal wrapper around the OwnerManager contract. This contract is meant to be initialized with
///      the same owners as a Safe instance, and then used to simulate the resulting owners list
///      after an owner is removed.
contract OwnerSimulator is OwnerManager {
    constructor(address[] memory _owners, uint256 _threshold) {
        setupOwners(_owners, _threshold);
    }

    /// @dev Exposes the OwnerManager's removeOwner function so that anyone may call without needing auth
    function removeOwnerWrapped(address prevOwner, address owner, uint256 _threshold) public {
        OwnerManager(address(this)).removeOwner(prevOwner, owner, _threshold);
    }
}

/// @dev collapsed interface that includes comapatibilityfallback handler calls
abstract contract DeployedSafe is GnosisSafe, CompatibilityFallbackHandler { }

struct AdvancedSafeInitParams {
    bool includeFallbackHandler;
    uint256 saltNonce;
    address setupModulesCall_to;
    bytes setupModulesCall_data;
    uint256 refundAmount;
    address refundToken;
    address payable refundReceiver;
    bytes initData;
}

struct SafeInstance {
    uint256 instanceId;
    uint256[] ownerPKs;
    address[] owners;
    uint256 threshold;
    DeployedSafe safe;
}

library SafeTestLib {
    /// @dev A wrapper for the full execTransaction method, if no signatures are provided it will
    ///         generate them for all owners.
    function execTransaction(
        SafeInstance memory instance,
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address refundReceiver,
        bytes memory signatures
    )
        internal
        returns (bool)
    {
        if (instance.owners.length == 0) {
            revert("SAFETEST: Instance not initialized. Call _setupSafe() to initialize a test safe");
        }

        bytes32 safeTxHash;
        {
            uint256 _nonce = instance.safe.nonce();
            safeTxHash = instance.safe.getTransactionHash({
                to: to,
                value: value,
                data: data,
                operation: operation,
                safeTxGas: safeTxGas,
                baseGas: baseGas,
                gasPrice: gasPrice,
                gasToken: gasToken,
                refundReceiver: refundReceiver,
                _nonce: _nonce
            });
        }

        if (signatures.length == 0) {
            for (uint256 i; i < instance.ownerPKs.length; ++i) {
                uint256 pk = instance.ownerPKs[i];
                (uint8 v, bytes32 r, bytes32 s) = Vm(VM_ADDR).sign(pk, safeTxHash);
                if (isSmartContractPK(pk)) {
                    v = 0;
                    address addr = decodeSmartContractWalletAsAddress(pk);
                    assembly {
                        r := addr
                    }
                    console.logBytes32(r);
                }
                signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
            }
        }

        return instance.safe.execTransaction({
            to: to,
            value: value,
            data: data,
            operation: operation,
            safeTxGas: safeTxGas,
            baseGas: baseGas,
            gasPrice: gasPrice,
            gasToken: gasToken,
            refundReceiver: payable(refundReceiver),
            signatures: signatures
        });
    }

    /// @dev Executes either a CALL or DELEGATECALL transaction.
    function execTransaction(
        SafeInstance memory instance,
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation
    )
        internal
        returns (bool)
    {
        return execTransaction(instance, to, value, data, operation, 0, 0, 0, address(0), address(0), "");
    }

    /// @dev Executes a CALL transaction.
    function execTransaction(
        SafeInstance memory instance,
        address to,
        uint256 value,
        bytes memory data
    )
        internal
        returns (bool)
    {
        return execTransaction(instance, to, value, data, Enum.Operation.Call, 0, 0, 0, address(0), address(0), "");
    }

    /// @dev Enables a module on the Safe.
    function enableModule(SafeInstance memory instance, address module) internal {
        execTransaction(
            instance,
            address(instance.safe),
            0,
            abi.encodeWithSelector(ModuleManager.enableModule.selector, module),
            Enum.Operation.Call,
            0,
            0,
            0,
            address(0),
            address(0),
            ""
        );
    }

    /// @dev Disables a module on the Safe.
    function disableModule(SafeInstance memory instance, address module) internal {
        (address[] memory modules,) = instance.safe.getModulesPaginated(SENTINEL_MODULES, 1000);
        address prevModule = SENTINEL_MODULES;
        bool moduleFound;
        for (uint256 i; i < modules.length; i++) {
            if (modules[i] == module) {
                moduleFound = true;
                break;
            }
            prevModule = modules[i];
        }
        if (!moduleFound) revert("SAFETESTTOOLS: cannot disable module that is not enabled");

        execTransaction(
            instance,
            address(instance.safe),
            0,
            abi.encodeWithSelector(ModuleManager.disableModule.selector, prevModule, module),
            Enum.Operation.Call,
            0,
            0,
            0,
            address(0),
            address(0),
            ""
        );
    }

    /// @dev Sets the guard address on the Safe. Unlike modules there can only be one guard, so
    ///      this method will remove the previous guard. If the guard is set to the 0 address, the
    ///      guard will be disabled.
    function setGuard(SafeInstance memory instance, address guard) internal {
        execTransaction(
            instance,
            address(instance.safe),
            0,
            abi.encodeWithSelector(GuardManager.setGuard.selector, guard),
            Enum.Operation.Call,
            0,
            0,
            0,
            address(0),
            address(0),
            ""
        );
    }

    /// @dev Signs message data using EIP1271: Standard Signature Validation Method for Contracts
    function EIP1271Sign(SafeInstance memory instance, bytes memory data) internal {
        address signMessageLib = address(new SignMessageLib());
        execTransaction({
            instance: instance,
            to: signMessageLib,
            value: 0,
            data: abi.encodeWithSelector(SignMessageLib.signMessage.selector, data),
            operation: Enum.Operation.DelegateCall,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(address(0)),
            signatures: ""
        });
    }

    /// @dev Signs a data hash using EIP1271: Standard Signature Validation Method for Contracts
    function EIP1271Sign(SafeInstance memory instance, bytes32 digest) internal {
        EIP1271Sign(instance, abi.encodePacked(digest));
    }

    /// @dev Sign a transaction as a safe owner with a private key.
    function signTransaction(
        SafeInstance memory instance,
        uint256 pk,
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address refundReceiver
    )
        internal
        view
        returns (uint8 v, bytes32 r, bytes32 s)
    {
        bytes32 txDataHash;
        {
            uint256 _nonce = instance.safe.nonce();
            txDataHash = instance.safe.getTransactionHash({
                to: to,
                value: value,
                data: data,
                operation: operation,
                safeTxGas: safeTxGas,
                baseGas: baseGas,
                gasPrice: gasPrice,
                gasToken: gasToken,
                refundReceiver: refundReceiver,
                _nonce: _nonce
            });
        }

        (v, r, s) = Vm(VM_ADDR).sign(pk, txDataHash);
    }

    /// @dev Increments the nonce of the Safe by sending an empty transaction.
    function incrementNonce(SafeInstance memory instance) internal returns (uint256 newNonce) {
        execTransaction(instance, address(0), 0, "", Enum.Operation.Call, 0, 0, 0, address(0), address(0), "");
        return instance.safe.nonce();
    }

    /// @notice Get the previous owner in the linked list of owners
    /// @param _owner The owner whose previous owner we want to find
    /// @param _owners The list of owners
    function getPrevOwner(address _owner, address[] memory _owners) internal pure returns (address prevOwner_) {
        for (uint256 i = 0; i < _owners.length; i++) {
            if (_owners[i] != _owner) continue;
            if (i == 0) {
                prevOwner_ = SENTINEL_OWNERS;
                break;
            }
            prevOwner_ = _owners[i - 1];
        }
    }

    /// @dev Given an array of owners to remove, this function will return an array of the previous owners
    ///         in the order that they must be provided to the LivenessMoules's removeOwners() function.
    ///         Because owners are removed one at a time, and not necessarily in order, we need to simulate
    ///         the owners list after each removal, in order to identify the correct previous owner.
    /// @param _ownersToRemove The owners to remove
    /// @return prevOwners_ The previous owners in the linked list
    function getPrevOwners(
        SafeInstance memory instance,
        address[] memory _ownersToRemove
    )
        internal
        returns (address[] memory prevOwners_)
    {
        OwnerSimulator ownerSimulator = new OwnerSimulator(instance.owners, 1);
        prevOwners_ = new address[](_ownersToRemove.length);
        address[] memory currentOwners;
        for (uint256 i = 0; i < _ownersToRemove.length; i++) {
            currentOwners = ownerSimulator.getOwners();
            prevOwners_[i] = SafeTestLib.getPrevOwner(instance.owners[i], currentOwners);

            // Don't try to remove the last owner
            if (currentOwners.length == 1) break;
            ownerSimulator.removeOwnerWrapped(prevOwners_[i], _ownersToRemove[i], 1);
        }
    }
}

/// @dev SafeTestTools implements a set of helper functions for testing Safe contracts.
contract SafeTestTools {
    using SafeTestLib for SafeInstance;

    GnosisSafe internal singleton = new GnosisSafe();
    GnosisSafeProxyFactory internal proxyFactory = new GnosisSafeProxyFactory();
    CompatibilityFallbackHandler internal handler = new CompatibilityFallbackHandler();

    SafeInstance[] internal instances;

    /// @dev can be called to reinitialize the singleton, proxyFactory and handler. Useful for forking.
    function _initializeSafeTools() internal {
        singleton = new GnosisSafe();
        proxyFactory = new GnosisSafeProxyFactory();
        handler = new CompatibilityFallbackHandler();
    }

    function _setupSafe(
        uint256[] memory ownerPKs,
        uint256 threshold,
        uint256 initialBalance,
        AdvancedSafeInitParams memory advancedParams
    )
        public
        returns (SafeInstance memory)
    {
        uint256[] memory sortedPKs = sortPKsByComputedAddress(ownerPKs);
        address[] memory owners = new address[](sortedPKs.length);

        for (uint256 i; i < sortedPKs.length; i++) {
            if (isSmartContractPK(sortedPKs[i])) {
                owners[i] = decodeSmartContractWalletAsAddress(sortedPKs[i]);
            } else {
                owners[i] = getAddr(sortedPKs[i]);
            }
        }
        // store the initialization parameters

        bytes memory initData = advancedParams.initData.length > 0
            ? advancedParams.initData
            : abi.encodeWithSelector(
                GnosisSafe.setup.selector,
                owners,
                threshold,
                advancedParams.setupModulesCall_to,
                advancedParams.setupModulesCall_data,
                advancedParams.includeFallbackHandler ? address(handler) : address(0),
                advancedParams.refundToken,
                advancedParams.refundAmount,
                advancedParams.refundReceiver
            );

        DeployedSafe safe0 = DeployedSafe(
            payable(proxyFactory.createProxyWithNonce(address(singleton), initData, advancedParams.saltNonce))
        );

        SafeInstance memory instance0 = SafeInstance({
            instanceId: instances.length,
            ownerPKs: sortedPKs,
            owners: owners,
            threshold: threshold,
            // setup safe ecosystem, singleton, proxy factory, fallback handler, and create a new safe
            safe: safe0
        });
        instances.push(instance0);

        Vm(VM_ADDR).deal(address(safe0), initialBalance);

        return instance0;
    }

    function _setupSafe(
        uint256[] memory ownerPKs,
        uint256 threshold,
        uint256 initialBalance
    )
        public
        returns (SafeInstance memory)
    {
        return _setupSafe(
            ownerPKs,
            threshold,
            initialBalance,
            AdvancedSafeInitParams({
                includeFallbackHandler: true,
                initData: "",
                saltNonce: 0,
                setupModulesCall_to: address(0),
                setupModulesCall_data: "",
                refundAmount: 0,
                refundToken: address(0),
                refundReceiver: payable(address(0))
            })
        );
    }

    function _setupSafe(uint256[] memory ownerPKs, uint256 threshold) public returns (SafeInstance memory) {
        return _setupSafe(
            ownerPKs,
            threshold,
            10000 ether,
            AdvancedSafeInitParams({
                includeFallbackHandler: true,
                initData: "",
                saltNonce: 0,
                setupModulesCall_to: address(0),
                setupModulesCall_data: "",
                refundAmount: 0,
                refundToken: address(0),
                refundReceiver: payable(address(0))
            })
        );
    }

    function _setupSafe() public returns (SafeInstance memory) {
        (, uint256[] memory defaultPKs) = makeAddrsAndKeys(3);

        return _setupSafe(
            defaultPKs,
            2,
            10000 ether,
            AdvancedSafeInitParams({
                includeFallbackHandler: true,
                initData: "",
                saltNonce: uint256(keccak256(bytes("SAFE TEST"))),
                setupModulesCall_to: address(0),
                setupModulesCall_data: "",
                refundAmount: 0,
                refundToken: address(0),
                refundReceiver: payable(address(0))
            })
        );
    }

    function getSafe() public view returns (SafeInstance memory) {
        if (instances.length == 0) {
            revert("SAFETESTTOOLS: Test Safe has not been deployed, use _setupSafe() calling safe()");
        }
        return instances[0];
    }

    function getSafe(address _safe) public view returns (SafeInstance memory) {
        for (uint256 i; i < instances.length; ++i) {
            if (address(instances[i].safe) == _safe) return instances[i];
        }
        revert("SAFETESTTOOLS: Safe instance not found");
    }
}

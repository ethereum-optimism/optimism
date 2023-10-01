// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0 <0.9.0;

import "forge-std/Test.sol";
import "solady/utils/LibSort.sol";
import "safe-contracts/GnosisSafe.sol";
import "safe-contracts/proxies/GnosisSafeProxyFactory.sol";
import "safe-contracts/examples/libraries/SignMessage.sol";
import "./CompatibilityFallbackHandler_1_3_0.sol";
import "safe-contracts/examples/libraries/SignMessage.sol";

address constant VM_ADDR = 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D;
bytes12 constant ADDR_MASK = 0xffffffffffffffffffffffff;

function getAddr(uint256 pk) pure returns (address) {
    return Vm(VM_ADDR).addr(pk);
}

function encodeSmartContractWalletAsPK(address addr) pure returns (uint256 encodedPK) {
    assembly {
        let addr_b32 := addr
        encodedPK := or(addr, ADDR_MASK)
    }
}

function decodeSmartContractWalletAsAddress(uint256 pk) pure returns (address decodedAddr) {
    assembly {
        let addr := shl(96, pk)
        decodedAddr := shr(96, addr)
    }
}

function isSmartContractPK(uint256 pk) pure returns (bool isEncoded) {
    assembly {
        isEncoded := eq(shr(160, pk), shr(160, ADDR_MASK))
    }
}

library Sort {
    function sort(address[] memory arr) public pure returns (address[] memory) {
        LibSort.sort(arr);
        return arr;
    }
}

function sortPKsByComputedAddress(uint256[] memory _pks) pure returns (uint256[] memory) {
    uint256[] memory sortedPKs = new uint256[](_pks.length);

    address[] memory addresses = new address[](_pks.length);
    bytes32[2][] memory accounts = new bytes32[2][](_pks.length);

    for (uint256 i; i < _pks.length; i++) {
        address signer = getAddr(_pks[i]);
        addresses[i] = signer;
        accounts[i][0] = bytes32(abi.encode(signer));
        accounts[i][1] = bytes32(_pks[i]);
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

// collapsed interface that includes comapatibilityfallback handler calls
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
        public
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

    function execTransaction(
        SafeInstance memory instance,
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation
    )
        public
        returns (bool)
    {
        return execTransaction(instance, to, value, data, operation, 0, 0, 0, address(0), address(0), "");
    }

    /// @dev performs a noraml "call"
    function execTransaction(
        SafeInstance memory instance,
        address to,
        uint256 value,
        bytes memory data
    )
        public
        returns (bool)
    {
        return execTransaction(instance, to, value, data, Enum.Operation.Call, 0, 0, 0, address(0), address(0), "");
    }

    function enableModule(SafeInstance memory instance, address module) public {
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

    function disableModule(SafeInstance memory instance, address module) public {
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

    function EIP1271Sign(SafeInstance memory instance, bytes memory data) public {
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

    function EIP1271Sign(SafeInstance memory instance, bytes32 digest) public {
        EIP1271Sign(instance, abi.encodePacked(digest));
    }

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
        public
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

    function incrementNonce(SafeInstance memory instance) public returns (uint256 newNonce) {
        execTransaction(instance, address(0), 0, "", Enum.Operation.Call, 0, 0, 0, address(0), address(0), "");
        return instance.safe.nonce();
    }
}

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
            payable(
                advancedParams.saltNonce != 0
                    ? proxyFactory.createProxyWithNonce(address(singleton), initData, advancedParams.saltNonce)
                    : proxyFactory.createProxy(address(singleton), initData)
            )
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
        string[3] memory users;
        users[0] = "SAFETEST: Signer 0";
        users[1] = "SAFETEST: Signer 1";
        users[2] = "SAFETEST: Signer 2";

        uint256[] memory defaultPKs = new uint256[](3);
        defaultPKs[0] = 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80;
        defaultPKs[1] = 0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d;
        defaultPKs[2] = 0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a;

        for (uint256 i; i < 3; i++) {
            Vm(VM_ADDR).label(getAddr(defaultPKs[i]), users[i]);
        }

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

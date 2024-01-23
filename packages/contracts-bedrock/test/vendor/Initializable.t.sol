// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Bridge_Initializer } from "test/setup/Bridge_Initializer.sol";
import { Executables } from "scripts/Executables.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import "src/L1/ProtocolVersions.sol";
import "scripts/Deployer.sol";

/// @title Initializer_Test
/// @dev Ensures that the `initialize()` function on contracts cannot be called more than
///      once. This contract inherits from `ERC721Bridge_Initializer` because it is the
///      deepest contract in the inheritance chain for setting up the system contracts.
///      For each L1 contract both the implementation and the proxy are tested.
contract Initializer_Test is Bridge_Initializer {
    /// @notice Contains the address of an `Initializable` contract and the calldata
    ///         used to initialize it.
    struct InitializeableContract {
        address target;
        bytes initCalldata;
        uint8 initializedSlotVal;
    }

    /// @notice Contains the addresses of the contracts to test as well as the calldata
    ///         used to initialize them.
    InitializeableContract[] contracts;

    function setUp() public override {
        // Run the `Bridge_Initializer`'s `setUp()` function.
        super.setUp();

        // Initialize the `contracts` array with the addresses of the contracts to test, the
        // calldata used to initialize them, and the storage slot of their `_initialized` flag.

        // SuperchainConfigImpl
        contracts.push(
            InitializeableContract({
                target: deploy.mustGetAddress("SuperchainConfig"),
                initCalldata: abi.encodeCall(superchainConfig.initialize, (address(0), false)),
                initializedSlotVal: deploy.loadInitializedSlot("SuperchainConfig")
            })
        );
        // SuperchainConfigProxy
        contracts.push(
            InitializeableContract({
                target: address(superchainConfig),
                initCalldata: abi.encodeCall(superchainConfig.initialize, (address(0), false)),
                initializedSlotVal: deploy.loadInitializedSlot("SuperchainConfigProxy")
            })
        );
        // L1CrossDomainMessengerImpl
        contracts.push(
            InitializeableContract({
                target: deploy.mustGetAddress("L1CrossDomainMessenger"),
                initCalldata: abi.encodeCall(l1CrossDomainMessenger.initialize, (superchainConfig, optimismPortal)),
                initializedSlotVal: deploy.loadInitializedSlot("L1CrossDomainMessenger")
            })
        );
        // L1CrossDomainMessengerProxy
        contracts.push(
            InitializeableContract({
                target: address(l1CrossDomainMessenger),
                initCalldata: abi.encodeCall(l1CrossDomainMessenger.initialize, (superchainConfig, optimismPortal)),
                initializedSlotVal: deploy.loadInitializedSlot("L1CrossDomainMessengerProxy")
            })
        );
        // L2OutputOracleImpl
        contracts.push(
            InitializeableContract({
                target: deploy.mustGetAddress("L2OutputOracle"),
                initCalldata: abi.encodeCall(l2OutputOracle.initialize, (0, 0, 0, 0, address(0), address(0), 0)),
                initializedSlotVal: deploy.loadInitializedSlot("L2OutputOracle")
            })
        );
        // L2OutputOracleProxy
        contracts.push(
            InitializeableContract({
                target: address(l2OutputOracle),
                initCalldata: abi.encodeCall(l2OutputOracle.initialize, (0, 0, 0, 0, address(0), address(0), 0)),
                initializedSlotVal: deploy.loadInitializedSlot("L2OutputOracleProxy")
            })
        );
        // OptimismPortalImpl
        contracts.push(
            InitializeableContract({
                target: deploy.mustGetAddress("OptimismPortal"),
                initCalldata: abi.encodeCall(optimismPortal.initialize, (l2OutputOracle, systemConfig, superchainConfig)),
                initializedSlotVal: deploy.loadInitializedSlot("OptimismPortal")
            })
        );
        // OptimismPortalProxy
        contracts.push(
            InitializeableContract({
                target: address(optimismPortal),
                initCalldata: abi.encodeCall(optimismPortal.initialize, (l2OutputOracle, systemConfig, superchainConfig)),
                initializedSlotVal: deploy.loadInitializedSlot("OptimismPortalProxy")
            })
        );
        // SystemConfigImpl
        contracts.push(
            InitializeableContract({
                target: deploy.mustGetAddress("SystemConfig"),
                initCalldata: abi.encodeCall(
                    systemConfig.initialize,
                    (
                        address(0xdead),
                        0,
                        0,
                        bytes32(0),
                        1,
                        address(0),
                        ResourceMetering.ResourceConfig({
                            maxResourceLimit: 1,
                            elasticityMultiplier: 1,
                            baseFeeMaxChangeDenominator: 2,
                            minimumBaseFee: 0,
                            systemTxMaxGas: 0,
                            maximumBaseFee: 0
                        }),
                        address(0),
                        SystemConfig.Addresses({
                            l1CrossDomainMessenger: address(0),
                            l1ERC721Bridge: address(0),
                            l1StandardBridge: address(0),
                            l2OutputOracle: address(0),
                            optimismPortal: address(0),
                            optimismMintableERC20Factory: address(0)
                        })
                    )
                    ),
                initializedSlotVal: deploy.loadInitializedSlot("SystemConfig")
            })
        );
        // SystemConfigProxy
        contracts.push(
            InitializeableContract({
                target: address(systemConfig),
                initCalldata: abi.encodeCall(
                    systemConfig.initialize,
                    (
                        address(0xdead),
                        0,
                        0,
                        bytes32(0),
                        1,
                        address(0),
                        ResourceMetering.ResourceConfig({
                            maxResourceLimit: 1,
                            elasticityMultiplier: 1,
                            baseFeeMaxChangeDenominator: 2,
                            minimumBaseFee: 0,
                            systemTxMaxGas: 0,
                            maximumBaseFee: 0
                        }),
                        address(0),
                        SystemConfig.Addresses({
                            l1CrossDomainMessenger: address(0),
                            l1ERC721Bridge: address(0),
                            l1StandardBridge: address(0),
                            l2OutputOracle: address(0),
                            optimismPortal: address(0),
                            optimismMintableERC20Factory: address(0)
                        })
                    )
                    ),
                initializedSlotVal: deploy.loadInitializedSlot("SystemConfigProxy")
            })
        );
        // ProtocolVersionsImpl
        contracts.push(
            InitializeableContract({
                target: deploy.mustGetAddress("ProtocolVersions"),
                initCalldata: abi.encodeCall(
                    protocolVersions.initialize, (address(0), ProtocolVersion.wrap(1), ProtocolVersion.wrap(2))
                    ),
                initializedSlotVal: deploy.loadInitializedSlot("ProtocolVersions")
            })
        );
        // ProtocolVersionsProxy
        contracts.push(
            InitializeableContract({
                target: address(protocolVersions),
                initCalldata: abi.encodeCall(
                    protocolVersions.initialize, (address(0), ProtocolVersion.wrap(1), ProtocolVersion.wrap(2))
                    ),
                initializedSlotVal: deploy.loadInitializedSlot("ProtocolVersionsProxy")
            })
        );
        // L2CrossDomainMessenger
        contracts.push(
            InitializeableContract({
                target: address(l2CrossDomainMessenger),
                initCalldata: abi.encodeCall(l2CrossDomainMessenger.initialize, (l1CrossDomainMessenger)),
                initializedSlotVal: deploy.loadInitializedSlot("L2CrossDomainMessenger")
            })
        );
        // L1StandardBridgeImpl
        contracts.push(
            InitializeableContract({
                target: deploy.mustGetAddress("L1StandardBridge"),
                initCalldata: abi.encodeCall(l1StandardBridge.initialize, (l1CrossDomainMessenger, superchainConfig)),
                initializedSlotVal: deploy.loadInitializedSlot("L1StandardBridge")
            })
        );
        // L1StandardBridgeProxy
        contracts.push(
            InitializeableContract({
                target: address(l1StandardBridge),
                initCalldata: abi.encodeCall(l1StandardBridge.initialize, (l1CrossDomainMessenger, superchainConfig)),
                initializedSlotVal: deploy.loadInitializedSlot("L1StandardBridgeProxy")
            })
        );
        // L2StandardBridge
        contracts.push(
            InitializeableContract({
                target: address(l2StandardBridge),
                initCalldata: abi.encodeCall(l2StandardBridge.initialize, (l1StandardBridge)),
                initializedSlotVal: deploy.loadInitializedSlot("L2StandardBridge")
            })
        );
        // L1ERC721BridgeImpl
        contracts.push(
            InitializeableContract({
                target: deploy.mustGetAddress("L1ERC721Bridge"),
                initCalldata: abi.encodeCall(l1ERC721Bridge.initialize, (l1CrossDomainMessenger, superchainConfig)),
                initializedSlotVal: deploy.loadInitializedSlot("L1ERC721Bridge")
            })
        );
        // L1ERC721BridgeProxy
        contracts.push(
            InitializeableContract({
                target: address(l1ERC721Bridge),
                initCalldata: abi.encodeCall(l1ERC721Bridge.initialize, (l1CrossDomainMessenger, superchainConfig)),
                initializedSlotVal: deploy.loadInitializedSlot("L1ERC721BridgeProxy")
            })
        );
        // L2ERC721Bridge
        contracts.push(
            InitializeableContract({
                target: address(l2ERC721Bridge),
                initCalldata: abi.encodeCall(l2ERC721Bridge.initialize, (payable(address(l1ERC721Bridge)))),
                initializedSlotVal: deploy.loadInitializedSlot("L2ERC721Bridge")
            })
        );
        // OptimismMintableERC20FactoryImpl
        contracts.push(
            InitializeableContract({
                target: deploy.mustGetAddress("OptimismMintableERC20Factory"),
                initCalldata: abi.encodeCall(l1OptimismMintableERC20Factory.initialize, (address(l1StandardBridge))),
                initializedSlotVal: deploy.loadInitializedSlot("OptimismMintableERC20Factory")
            })
        );
        // OptimismMintableERC20FactoryProxy
        contracts.push(
            InitializeableContract({
                target: address(l1OptimismMintableERC20Factory),
                initCalldata: abi.encodeCall(l1OptimismMintableERC20Factory.initialize, (address(l1StandardBridge))),
                initializedSlotVal: deploy.loadInitializedSlot("OptimismMintableERC20FactoryProxy")
            })
        );
    }

    /// @notice Tests that:
    ///         1. All `Initializable` contracts in `src/L1` and `src/L2` are accounted for in the `contracts` array.
    ///         2. The `_initialized` flag of each contract is properly set to `1`, signifying that the
    ///            contracts are initialized.
    ///         3. The `initialize()` function of each contract cannot be called more than once.
    function test_cannotReinitialize_succeeds() public {
        // Ensure that all L1, L2 `Initializable` contracts are accounted for, in addition to
        // OptimismMintableERC20FactoryImpl and OptimismMintableERC20FactoryProxy
        assertEq(_getNumInitializable() + 2, contracts.length);

        // Attempt to re-initialize all contracts within the `contracts` array.
        for (uint256 i; i < contracts.length; i++) {
            InitializeableContract memory _contract = contracts[i];
            uint256 size;
            address target = _contract.target;
            assembly {
                size := extcodesize(target)
            }
            // Assert that the contract is already initialized.
            assertEq(_contract.initializedSlotVal, 1);

            // Then, attempt to re-initialize the contract. This should fail.
            (bool success, bytes memory returnData) = _contract.target.call(_contract.initCalldata);
            assertFalse(success);
            assertEq(_extractErrorString(returnData), "Initializable: contract is already initialized");
        }
    }

    /// @dev Returns the number of contracts that are `Initializable` in `src/L1` and `src/L2`.
    ///      For L1 contracts, implementations are considered in addition to proxies
    function _getNumInitializable() internal returns (uint256 numContracts_) {
        string[] memory command = new string[](3);
        command[0] = Executables.bash;
        command[1] = "-c";
        // Start by getting L1 contracts
        command[2] = string.concat(
            Executables.find,
            " src/L1 -type f -exec basename {} \\;",
            " | ",
            Executables.sed,
            " 's/\\.[^.]*$//'",
            " | ",
            Executables.jq,
            " -R -s 'split(\"\n\")[:-1]'"
        );
        string[] memory l1ContractNames = abi.decode(vm.parseJson(string(vm.ffi(command))), (string[]));

        for (uint256 i; i < l1ContractNames.length; i++) {
            string memory contractName = l1ContractNames[i];
            string memory contractAbi = deploy.getAbi(contractName);

            // Query the contract's ABI for an `initialize()` function.
            command[2] = string.concat(
                Executables.echo,
                " '",
                contractAbi,
                "'",
                " | ",
                Executables.jq,
                " '.[] | select(.name == \"initialize\" and .type == \"function\")'"
            );
            bytes memory res = vm.ffi(command);

            // If the contract has an `initialize()` function, the resulting query will be non-empty.
            // In this case, increment the number of `Initializable` contracts.
            if (res.length > 0) {
                // Count Proxy + Impl
                numContracts_ += 2;
            }
        }

        // Then get L2 contracts
        command[2] = string.concat(
            Executables.find,
            " src/L2 -type f -exec basename {} \\;",
            " | ",
            Executables.sed,
            " 's/\\.[^.]*$//'",
            " | ",
            Executables.jq,
            " -R -s 'split(\"\n\")[:-1]'"
        );
        string[] memory l2ContractNames = abi.decode(vm.parseJson(string(vm.ffi(command))), (string[]));

        for (uint256 i; i < l2ContractNames.length; i++) {
            string memory contractName = l2ContractNames[i];
            string memory contractAbi = deploy.getAbi(contractName);

            // Query the contract's ABI for an `initialize()` function.
            command[2] = string.concat(
                Executables.echo,
                " '",
                contractAbi,
                "'",
                " | ",
                Executables.jq,
                " '.[] | select(.name == \"initialize\" and .type == \"function\")'"
            );
            bytes memory res = vm.ffi(command);

            // If the contract has an `initialize()` function, the resulting query will be non-empty.
            // In this case, increment the number of `Initializable` contracts.
            if (res.length > 0) {
                numContracts_++;
            }
        }
    }

    /// @dev Extracts the revert string from returndata encoded in the form of `Error(string)`.
    function _extractErrorString(bytes memory _returnData) internal pure returns (string memory error_) {
        // The first 4 bytes of the return data should be the selector for `Error(string)`. If not, revert.
        if (bytes4(_returnData) == 0x08c379a0) {
            // Extract the error string from the returndata. The error string is located 68 bytes after
            // the pointer to `returnData`.
            //
            // 32 bytes: `returnData` length
            // 4 bytes: `Error(string)` selector
            // 32 bytes: ABI encoding metadata; String offset
            // = 68 bytes
            assembly {
                error_ := add(_returnData, 0x44)
            }
        } else {
            revert("Initializer_Test: Invalid returndata format. Expected `Error(string)`");
        }
    }
}

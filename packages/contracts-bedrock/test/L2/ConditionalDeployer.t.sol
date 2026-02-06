// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "forge-std/Test.sol";

// Libraries
import { Config } from "scripts/libraries/Config.sol";

// Contracts
import { ConditionalDeployer } from "src/L2/ConditionalDeployer.sol";

/// @title ConditionalDeployer_Harness
/// @notice A simple contract harness used for deployment testing of the ConditionalDeployer.
contract ConditionalDeployer_Harness {
    uint256 public immutable value;

    constructor(uint256 _value) {
        value = _value;
    }
}

/// @title ConditionalDeployer_TestInit
/// @notice Reusable test initialization for `ConditionalDeployer` tests.
contract ConditionalDeployer_TestInit is Test {
    // Test contracts
    ConditionalDeployer public conditionalDeployer;
    bytes public simpleContractCreationCode;

    function setUp() public {
        // Create fork
        vm.createSelectFork(Config.forkRpcUrl());

        // Deploy contracts
        conditionalDeployer = new ConditionalDeployer();
        simpleContractCreationCode = type(ConditionalDeployer_Harness).creationCode;
    }
}

/// @title ConditionalDeployer_Deploy_Test
/// @notice Tests the `deploy` function of the `ConditionalDeployer` contract.
contract ConditionalDeployer_Deploy_Test is ConditionalDeployer_TestInit {
    /// @notice Event emitted when an implementation is deployed.
    event ImplementationDeployed(address indexed implementation, bytes32 salt);

    /// @notice Event emitted when deployment is skipped because implementation already exists.
    event ImplementationExists(address indexed implementation);

    /// @notice Tests that `deploy` succeeds and emits the correct event.
    function testFuzz_deploy_succeeds(address _caller, bytes32 _salt, uint256 _value) public {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(_value));
        bytes32 codeHash = keccak256(_initCode);
        address expectedImplementation = address(
            uint160(
                uint256(
                    keccak256(
                        abi.encodePacked(
                            bytes1(0xff), conditionalDeployer.DETERMINISTIC_DEPLOYMENT_PROXY(), _salt, codeHash
                        )
                    )
                )
            )
        );

        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationDeployed(expectedImplementation, _salt);

        vm.prank(_caller);
        address implementation = conditionalDeployer.deploy(0, _salt, _initCode);

        assertEq(implementation, expectedImplementation);
        assertEq(ConditionalDeployer_Harness(implementation).value(), _value);
        assert(implementation.code.length != 0);
    }

    /// @notice Tests that `deploy` is idempotent and produces the same address when called multiple times.
    function testFuzz_deploy_idempotent_succeeds(address _caller, bytes32 _salt, uint256 _value) public {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(_value));

        // First Deployment
        bytes32 codeHash = keccak256(_initCode);
        address expectedImplementation = address(
            uint160(
                uint256(
                    keccak256(
                        abi.encodePacked(
                            bytes1(0xff), conditionalDeployer.DETERMINISTIC_DEPLOYMENT_PROXY(), _salt, codeHash
                        )
                    )
                )
            )
        );

        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationDeployed(expectedImplementation, _salt);

        vm.prank(_caller);
        address implementation1 = conditionalDeployer.deploy(0, _salt, _initCode);

        // Assert that the implementation was deployed
        assert(implementation1.code.length != 0);

        // Second Deployment
        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationExists(implementation1);

        vm.prank(_caller);
        address implementation2 = conditionalDeployer.deploy(0, _salt, _initCode);

        assertEq(implementation1, implementation2);
    }

    /// @notice Tests that `deploy` reverts when the deployment call to the DeterministicDeploymentProxy fails.
    /// @dev The deployment call to the DeterministicDeploymentProxy is mocked to revert.
    function testFuzz_deploy_deploymentFailed_reverts(address _caller, bytes32 _salt, uint256 _value) public {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(0));

        vm.mockCallRevert(
            conditionalDeployer.DETERMINISTIC_DEPLOYMENT_PROXY(), _value, abi.encodePacked(_salt, _initCode), bytes("")
        );

        vm.prank(_caller);
        vm.expectRevert(
            abi.encodeWithSelector(ConditionalDeployer.ConditionalDeployer_DeploymentFailed.selector, bytes(""))
        );
        conditionalDeployer.deploy(_value, _salt, _initCode);
    }
}

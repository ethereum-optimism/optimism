// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";

// Contracts
import { ConditionalDeployer } from "src/L2/ConditionalDeployer.sol";

/// @title ConditionalDeployer_Harness
/// @notice A simple contract harness used for deployment testing of the ConditionalDeployer.
contract ConditionalDeployer_Harness {
    uint256 public immutable value;

    constructor(uint256 _value) payable {
        value = _value;
    }
}

/// @title ConditionalDeployer_TestInit
/// @notice Reusable test initialization for `ConditionalDeployer` tests.
contract ConditionalDeployer_TestInit is CommonTest {
    // Test contracts
    bytes public simpleContractCreationCode;

    function setUp() public override {
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.L2CM);
        // Deploy contracts
        simpleContractCreationCode = type(ConditionalDeployer_Harness).creationCode;
    }
}

/// @title ConditionalDeployer_Getters_Test
/// @notice Tests the getter functions of the `ConditionalDeployer` contract.
contract ConditionalDeployer_Version_Test is ConditionalDeployer_TestInit {
    /// @notice Tests that the version function returns a valid string.
    function test_version_succeeds() external view {
        assert(bytes(conditionalDeployer.version()).length > 0);
    }
}

/// @title ConditionalDeployer_DeterministicDeploymentProxy_Test
/// @notice Tests the deterministicDeploymentProxy function of the `ConditionalDeployer` contract.
contract ConditionalDeployer_DeterministicDeploymentProxy_Test is ConditionalDeployer_TestInit {
    /// @notice Tests that the deterministicDeploymentProxy function returns the correct address.
    function test_deterministicDeploymentProxy_succeeds() external view {
        assertEq(conditionalDeployer.deterministicDeploymentProxy(), payable(Preinstalls.DeterministicDeploymentProxy));
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
                            bytes1(0xff), conditionalDeployer.deterministicDeploymentProxy(), _salt, codeHash
                        )
                    )
                )
            )
        );

        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationDeployed(expectedImplementation, _salt);

        vm.prank(_caller);
        address implementation = conditionalDeployer.deploy(_salt, _initCode);

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
                            bytes1(0xff), conditionalDeployer.deterministicDeploymentProxy(), _salt, codeHash
                        )
                    )
                )
            )
        );

        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationDeployed(expectedImplementation, _salt);

        vm.prank(_caller);
        address implementation1 = conditionalDeployer.deploy(_salt, _initCode);

        // Assert that the implementation was deployed
        assert(implementation1.code.length != 0);

        // Second Deployment
        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationExists(implementation1);

        vm.prank(_caller);
        address implementation2 = conditionalDeployer.deploy(_salt, _initCode);

        assertEq(implementation1, implementation2);
    }

    /// @notice Tests that `deploy` reverts when the deployment call to the DeterministicDeploymentProxy fails.
    /// @dev The deployment call to the DeterministicDeploymentProxy is mocked to revert.
    function testFuzz_deploy_deploymentFailed_reverts(address _caller, bytes32 _salt) public {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(0));

        // Mock the deployment call to the DeterministicDeploymentProxy to revert
        vm.mockCallRevert(
            conditionalDeployer.deterministicDeploymentProxy(),
            0,
            abi.encodePacked(_salt, _initCode),
            bytes("deployment failed")
        );

        vm.prank(_caller);
        vm.expectRevert(
            abi.encodeWithSelector(
                ConditionalDeployer.ConditionalDeployer_DeploymentFailed.selector, bytes("deployment failed")
            )
        );
        conditionalDeployer.deploy(_salt, _initCode);
    }

    /// @notice Tests that `deploy` reverts when the deployment call to the DeterministicDeploymentProxy returns the
    /// wrong address.
    /// @dev The deployment call to the DeterministicDeploymentProxy is mocked to return the wrong address.
    function testFuzz_deploy_notExpectedAddress_reverts(
        address _caller,
        bytes32 _salt,
        address _notExpectedAddress,
        uint256 _value
    )
        public
    {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(_value));
        bytes32 codeHash = keccak256(_initCode);
        address expectedImplementation = address(
            uint160(
                uint256(
                    keccak256(
                        abi.encodePacked(
                            bytes1(0xff), conditionalDeployer.deterministicDeploymentProxy(), _salt, codeHash
                        )
                    )
                )
            )
        );
        vm.assume(_notExpectedAddress != expectedImplementation);

        vm.mockCall(
            conditionalDeployer.deterministicDeploymentProxy(),
            0,
            abi.encodePacked(_salt, _initCode),
            abi.encodePacked(_notExpectedAddress)
        );
        vm.prank(_caller);
        vm.expectRevert(
            abi.encodeWithSelector(
                ConditionalDeployer.ConditionalDeployer_DeploymentFailed.selector, abi.encodePacked(_notExpectedAddress)
            )
        );
        conditionalDeployer.deploy(_salt, _initCode);
    }
}

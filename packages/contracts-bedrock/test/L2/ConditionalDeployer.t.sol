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
/// @notice This contract is deployed by the ConditionalDeployer to test the deployment of an
/// implementation.
contract ConditionalDeployer_Harness {
    uint256 public immutable number;

    constructor(uint256 _number) {
        number = _number;
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

/// @title ConditionalDeployer_Deploy_Test
/// @notice Tests the `deploy` function of the `ConditionalDeployer` contract.
contract ConditionalDeployer_Deploy_Test is ConditionalDeployer_TestInit {
    /// @notice Event emitted when an implementation is deployed.
    event ImplementationDeployed(address indexed implementation, bytes32 salt);

    /// @notice Event emitted when deployment is skipped because implementation already exists.
    event ImplementationExists(address indexed implementation);

    /// @notice Tests that `deploy` succeeds and emits the correct event.
    function testFuzz_deploy_succeeds(address _caller, bytes32 _salt, uint256 _number) public {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(_number));
        address expectedImplementation = getExpectedImplementation(_initCode, _salt);

        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationDeployed(expectedImplementation, _salt);

        vm.prank(_caller);
        address implementation = conditionalDeployer.deploy(_salt, _initCode);

        assertEq(implementation, expectedImplementation);
        assertEq(ConditionalDeployer_Harness(implementation).number(), _number);
        assert(implementation.code.length != 0);
    }

    /// @notice Tests that `deploy` is idempotent and produces the same address when called multiple times.
    function testFuzz_deploy_idempotent_succeeds(address _caller, bytes32 _salt, uint256 _number) public {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(_number));
        address expectedImplementation = getExpectedImplementation(_initCode, _salt);

        // First Deployment
        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationDeployed(expectedImplementation, _salt);

        assertEq(expectedImplementation.code.length, 0);

        vm.prank(_caller);
        address implementation1 = conditionalDeployer.deploy(_salt, _initCode);

        // Assert that the implementation was deployed
        assertEq(implementation1, expectedImplementation);
        assert(implementation1.code.length != 0);
        assertEq(ConditionalDeployer_Harness(implementation1).number(), _number);

        // Second Deployment
        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationExists(implementation1);

        vm.prank(_caller);
        address implementation2 = conditionalDeployer.deploy(_salt, _initCode);

        assertEq(implementation1, implementation2);
    }

    /// @notice Tests that `deploy` reverts when the deployment call to the DeterministicDeploymentProxy fails.
    /// @dev The deployment call to the DeterministicDeploymentProxy is mocked to revert.
    function testFuzz_deploy_deploymentFailed_reverts(address _caller, bytes32 _salt, uint256 _number) public {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(_number));

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
        uint256 _number
    )
        public
    {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(_number));
        address expectedImplementation = getExpectedImplementation(_initCode, _salt);
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

    /// @notice Returns the expected implementation address for the given initialization code and salt.
    /// @dev Uses the CREATE2 formula to compute the expected implementation address.
    /// @param _initCode The initialization code for the contract.
    /// @param _salt The salt to use for deployment.
    /// @return expectedImplementation_ The expected implementation address.
    function getExpectedImplementation(
        bytes memory _initCode,
        bytes32 _salt
    )
        internal
        view
        returns (address expectedImplementation_)
    {
        bytes32 codeHash = keccak256(_initCode);
        expectedImplementation_ = address(
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
    }
}

/// @title ConditionalDeployer_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `ConditionalDeployer`
///         contract or are testing multiple functions.
contract ConditionalDeployer_Uncategorized_Test is ConditionalDeployer_TestInit {
    /// @notice Tests that the getters return valid values.
    function test_getters_succeeds() external view {
        assert(bytes(conditionalDeployer.version()).length > 0);
        assertEq(conditionalDeployer.deterministicDeploymentProxy(), payable(Preinstalls.DeterministicDeploymentProxy));
    }
}

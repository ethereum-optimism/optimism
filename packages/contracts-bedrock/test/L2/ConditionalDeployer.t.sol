// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { ConditionalDeployer } from "src/L2/ConditionalDeployer.sol";
import { Config } from "scripts/libraries/Config.sol";
import { Constants } from "src/libraries/Constants.sol";
import { ICreate2Deployer } from "interfaces/preinstalls/ICreate2Deployer.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";

/// @title SimpleContract
/// @notice A simple contract to deploy using the ConditionalDeployer.
contract SimpleContract {
    uint256 public immutable value;

    constructor(uint256 _value) {
        value = _value;
    }
}

/// @title ConditionalDeployer_TestInit
/// @notice Reusable test initialization for `ConditionalDeployer` tests.
contract ConditionalDeployer_TestInit is Test {
    ConditionalDeployer public conditionalDeployer;
    bytes public simpleContractCreationCode;

    function setUp() public {
        vm.createSelectFork(Config.forkRpcUrl(), Config.forkBlockNumber());
        conditionalDeployer = new ConditionalDeployer();
        simpleContractCreationCode = type(SimpleContract).creationCode;
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
    function testFuzz_deploy_succeeds(bytes32 _salt, uint256 _value) public {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(_value));
        bytes32 codeHash = keccak256(_initCode);
        address expectedImplementation =
            ICreate2Deployer(payable(Preinstalls.Create2Deployer)).computeAddress(_salt, codeHash);

        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationDeployed(expectedImplementation, _salt);

        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        address implementation = conditionalDeployer.deploy(0, _salt, _initCode);

        assertEq(implementation, expectedImplementation);
        assertEq(SimpleContract(implementation).value(), _value);
        assert(implementation.code.length != 0);
    }

    /// @notice Tests that `deploy` succeeds when called by `address(0)`.
    function testFuzz_deploy_fromAddressZero_succeeds(bytes32 _salt, uint256 _value) public {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(_value));

        vm.prank(address(0));
        address implementation = conditionalDeployer.deploy(0, _salt, _initCode);

        assertEq(SimpleContract(implementation).value(), _value);
        assert(implementation.code.length != 0);
    }

    /// @notice Tests that `deploy` produces the same address when called multiple times.
    function testFuzz_deploy_produces_same_address_succeeds(bytes32 _salt, uint256 _value) public {
        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(_value));

        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        address implementation1 = conditionalDeployer.deploy(0, _salt, _initCode);

        // Assert that the implementation was deployed
        assert(implementation1.code.length != 0);

        // Attempt to deploy the same implementation again
        vm.expectEmit(address(conditionalDeployer));
        emit ImplementationExists(implementation1);

        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        address implementation2 = conditionalDeployer.deploy(0, _salt, _initCode);

        assertEq(implementation1, implementation2);
    }
}

/// @title ConditionalDeployer_Deploy_TestFail
/// @notice Tests failure cases for the `deploy` function of the `ConditionalDeployer` contract.
contract ConditionalDeployer_Deploy_TestFail is ConditionalDeployer_TestInit {
    /// @notice Tests that `deploy` reverts when called by an address other than the depositor account or address(0).
    function testFuzz_deploy_when_not_authorized_reverts(address _sender) public {
        vm.assume(_sender != Constants.DEPOSITOR_ACCOUNT && _sender != address(0));

        bytes memory _initCode = abi.encodePacked(simpleContractCreationCode, abi.encode(0));

        vm.prank(_sender);
        vm.expectRevert(ConditionalDeployer.UnauthorizedCaller.selector);
        conditionalDeployer.deploy(0, bytes32(0), _initCode);
    }
}

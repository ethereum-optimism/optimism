// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";

interface IEIP712 {
    function DOMAIN_SEPARATOR() external view returns (bytes32);
}

/// @title PredeploysTest
contract PredeploysTest is CommonTest {
    /// @dev The domain separator commits to the chainid of the chain
    function test_preinstall_permit2_succeeds(uint64 _chainId) external {
        vm.chainId(_chainId);

        bytes32 domainSeparator = IEIP712(Preinstalls.PERMIT2).DOMAIN_SEPARATOR();
        bytes32 typeHash =
            keccak256(abi.encodePacked("EIP712Domain(string name,uint256 chainId,address verifyingContract)"));
        bytes32 nameHash = keccak256(abi.encodePacked("Permit2"));
        bytes memory encoded = abi.encode(typeHash, nameHash, _chainId, Preinstalls.PERMIT2);
        bytes32 expectedDomainSeparator = keccak256(encoded);
        assertEq(domainSeparator, expectedDomainSeparator, "Domain separator mismatch");
    }

    //////////////////////////////////////////////////////
    /// Code Assertion Tests
    //////////////////////////////////////////////////////
    function test_preinstall_multicall3_succeeds() external {
        assertEq(Preinstalls.MULTICALL3.code, Preinstalls.MULTICALL3_DEPLOYED_BYTECODE);
    }

    function test_preinstall_create2Deployer_succeeds() external {
        assertEq(Preinstalls.CREATE2_DEPLOYER.code, Preinstalls.CREATE2_DEPLOYER_DEPLOYED_BYTECODE);
    }

    function test_preinstall_safev130_succeeds() external {
        assertEq(Preinstalls.SAFE_V130.code, Preinstalls.SAFE_V130_DEPLOYED_BYTECODE);
    }

    function test_preinstall_safeL2v130_succeeds() external {
        assertEq(Preinstalls.SAFE_L2_V130.code, Preinstalls.SAFE_L2_V130_DEPLOYED_BYTECODE);
    }

    function test_preinstall_multisendv130_succeeds() external {
        assertEq(Preinstalls.MULTI_SEND_V130.code, Preinstalls.MULTI_SEND_V130_DEPLOYED_BYTECODE);
    }

    function test_preinstall_multisendCallOnlyv130_succeeds() external {
        assertEq(Preinstalls.MULTI_SEND_CALL_ONLY_V130.code, Preinstalls.MULTI_SEND_CALL_ONLY_V130_DEPLOYED_BYTECODE);
    }

    function test_preinstall_safeSingletonFactory_succeeds() external {
        assertEq(Preinstalls.SAFE_SINGLETON_FACTORY.code, Preinstalls.SAFE_SINGLETON_FACTORY_DEPLOYED_BYTECODE);
    }

    function test_preinstall_deterministicDeploymentProxy_succeeds() external {
        assertEq(
            Preinstalls.DETERMINISTIC_DEPLOYMENT_PROXY.code,
            Preinstalls.DETERMINISTIC_DEPLOYMENT_PROXY_DEPLOYED_BYTECODE
        );
    }

    function test_preinstall_senderCreator_succeeds() external {
        assertEq(Preinstalls.SENDER_CREATOR.code, Preinstalls.SENDER_CREATOR_DEPLOYED_BYTECODE);
    }

    function test_preinstall_entrypoint_succeeds() external {
        assertEq(Preinstalls.ENTRY_POINT.code, Preinstalls.ENTRY_POINT_DEPLOYED_BYTECODE);
    }
}

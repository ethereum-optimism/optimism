// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IChainConfig } from "interfaces/L2/IChainConfig.sol";

/// @title ChainConfig_TestInit
/// @notice Places the ChainConfig code at its predeploy address, as genesis does.
abstract contract ChainConfig_TestInit is Test {
    IChainConfig internal chainConfig;

    /// @notice Test setup.
    function setUp() public virtual {
        vm.etch(Predeploys.CHAIN_CONFIG, vm.getDeployedCode("ChainConfig.sol:ChainConfig"));
        chainConfig = IChainConfig(Predeploys.CHAIN_CONFIG);
    }
}

/// @title ChainConfig_Version_Test
/// @notice Tests the `version` function of the `ChainConfig` contract.
contract ChainConfig_Version_Test is ChainConfig_TestInit {
    /// @notice Tests that the version is 1.0.0.
    function test_version_succeeds() external view {
        assertEq(chainConfig.version(), "1.0.0");
    }
}

/// @title ChainConfig_SettlementChainId_Test
/// @notice Tests the `settlementChainId` function of the `ChainConfig` contract.
contract ChainConfig_SettlementChainId_Test is ChainConfig_TestInit {
    /// @notice Tests that the settlement chain ID is 0 when genesis wrote none.
    function test_settlementChainId_unset_succeeds() external view {
        assertEq(chainConfig.settlementChainId(), 0);
    }

    /// @notice Tests that the settlement chain ID is the value genesis wrote into its slot.
    function testFuzz_settlementChainId_writtenInGenesis_succeeds(uint256 _id) external {
        vm.store(Predeploys.CHAIN_CONFIG, chainConfig.SETTLEMENT_CHAIN_ID_SLOT(), bytes32(_id));
        assertEq(chainConfig.settlementChainId(), _id);
    }

    /// @notice Tests that the storage slot is the documented one.
    function test_settlementChainId_slotMatchesDerivation_succeeds() external view {
        assertEq(
            chainConfig.SETTLEMENT_CHAIN_ID_SLOT(), bytes32(uint256(keccak256("chainconfig.settlementchainid")) - 1)
        );
    }
}

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

    /// @notice Writes the settlement chain ID, as genesis does.
    function _setSettlementChainId(uint256 _id) internal {
        vm.store(Predeploys.CHAIN_CONFIG, chainConfig.SETTLEMENT_CHAIN_ID_SLOT(), bytes32(_id));
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
        _setSettlementChainId(_id);
        assertEq(chainConfig.settlementChainId(), _id);
    }

    /// @notice Tests that the storage slot is the documented one.
    function test_settlementChainId_slotMatchesDerivation_succeeds() external view {
        assertEq(
            chainConfig.SETTLEMENT_CHAIN_ID_SLOT(), bytes32(uint256(keccak256("chainconfig.settlementchainid")) - 1)
        );
    }
}

/// @title ChainConfig_IsSettlementChain_Test
/// @notice Tests the `isSettlementChain` function of the `ChainConfig` contract.
contract ChainConfig_IsSettlementChain_Test is ChainConfig_TestInit {
    /// @notice Tests that a public chain, with no settlement chain ID, is not a settlement chain.
    function testFuzz_isSettlementChain_publicChain_succeeds(uint64 _chainId) external {
        vm.assume(_chainId != 0);
        vm.chainId(_chainId);
        assertFalse(chainConfig.isSettlementChain());
    }

    /// @notice Tests that a chain whose settlement chain ID is its own chain ID is a settlement
    ///         chain.
    function testFuzz_isSettlementChain_ownChainId_succeeds(uint64 _chainId) external {
        vm.assume(_chainId != 0);
        vm.chainId(_chainId);
        _setSettlementChainId(_chainId);
        assertTrue(chainConfig.isSettlementChain());
    }

    /// @notice Tests that a private chain is not a settlement chain.
    function testFuzz_isSettlementChain_privateChain_succeeds(uint64 _chainId, uint256 _settlementChainId) external {
        vm.assume(_chainId != 0 && _settlementChainId != 0 && _settlementChainId != _chainId);
        vm.chainId(_chainId);
        _setSettlementChainId(_settlementChainId);
        assertFalse(chainConfig.isSettlementChain());
    }
}

/// @title ChainConfig_IsPrivateChain_Test
/// @notice Tests the `isPrivateChain` function of the `ChainConfig` contract.
contract ChainConfig_IsPrivateChain_Test is ChainConfig_TestInit {
    /// @notice Tests that a public chain, with no settlement chain ID, is not a private chain.
    function testFuzz_isPrivateChain_publicChain_succeeds(uint64 _chainId) external {
        vm.assume(_chainId != 0);
        vm.chainId(_chainId);
        assertFalse(chainConfig.isPrivateChain());
    }

    /// @notice Tests that a settlement chain is not a private chain.
    function testFuzz_isPrivateChain_settlementChain_succeeds(uint64 _chainId) external {
        vm.assume(_chainId != 0);
        vm.chainId(_chainId);
        _setSettlementChainId(_chainId);
        assertFalse(chainConfig.isPrivateChain());
    }

    /// @notice Tests that a chain whose settlement chain ID is another chain's ID is a private
    ///         chain.
    function testFuzz_isPrivateChain_otherChainId_succeeds(uint64 _chainId, uint256 _settlementChainId) external {
        vm.assume(_chainId != 0 && _settlementChainId != 0 && _settlementChainId != _chainId);
        vm.chainId(_chainId);
        _setSettlementChainId(_settlementChainId);
        assertTrue(chainConfig.isPrivateChain());
    }
}

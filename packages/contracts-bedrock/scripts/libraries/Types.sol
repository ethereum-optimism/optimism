// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Claim, Duration, GameType } from "src/dispute/lib/Types.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

library Types {
    /// @notice Represents a set of L1 contracts. Used to represent a set of proxies.
    /// This is not an exhaustive list of all contracts on L1, but rather a subset.
    struct ContractSet {
        address L1CrossDomainMessenger;
        address L1StandardBridge;
        address L2OutputOracle;
        address DisputeGameFactory;
        address DelayedWETH;
        address PermissionedDelayedWETH;
        address AnchorStateRegistry;
        address OptimismMintableERC20Factory;
        address OptimismPortal;
        address ETHLockbox;
        address SystemConfig;
        address L1ERC721Bridge;
        address ProtocolVersions;
        address SuperchainConfig;
    }

    struct DeployOPChainInput {
        // Roles
        address opChainProxyAdminOwner;
        address systemConfigOwner;
        address batcher;
        address unsafeBlockSigner;
        address proposer;
        address challenger;
        // TODO Add fault proofs inputs in a future PR.
        uint32 basefeeScalar;
        uint32 blobBaseFeeScalar;
        uint256 l2ChainId;
        address opcm;
        string saltMixer;
        uint64 gasLimit;
        // Configurable dispute game inputs
        GameType disputeGameType;
        Claim disputeAbsolutePrestate;
        uint256 disputeMaxGameDepth;
        uint256 disputeSplitDepth;
        Duration disputeClockExtension;
        Duration disputeMaxClockDuration;
        bool allowCustomDisputeParameters;
        // Fee params
        uint32 operatorFeeScalar;
        uint64 operatorFeeConstant;
        // Superchain contracts
        ISuperchainConfig superchainConfig;
        // Whether to use the custom gas token.
        bool useCustomGasToken;
    }
}

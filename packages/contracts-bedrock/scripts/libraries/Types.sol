// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Claim, Duration, GameType, Proposal } from "src/dispute/lib/Types.sol";
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
        address SuperchainConfig;
    }

    /// @notice Overrides for the L1 resource config written at SystemConfig initialization.
    /// @dev The resource config is set ONLY through `SystemConfig.initialize` -- this version has no
    ///      owner-callable setter -- so a chain that wants a non-default one has to say so at deploy
    ///      time, and a live SystemConfig owner can still rotate the batcher key without being able
    ///      to change it back.
    ///
    ///      The reason this exists: a chain can be made undepositable with a stock portal by
    ///      initializing with `maxResourceLimit = 0`. Every `depositTransaction` is metered against
    ///      that limit with a gas limit of at least 21000, so every deposit reverts on L1, and
    ///      derivation and attribute handling are never touched because no user deposit can exist
    ///      for them to see. That matters for a private interop pair, whose ETH solvency story
    ///      depends on the public rendering never minting ETH from a deposit.
    ///
    ///      The three fields here are the ones `SystemConfig._setResourceConfig` constrains against
    ///      each other; the EIP-1559 parameters and base fee bounds keep their defaults, since
    ///      nothing about the deposit gate wants them moved.
    /// @custom:field enabled              False keeps the gas-limit-derived default, which is what
    ///                                    every ordinary chain uses.
    /// @custom:field maxResourceLimit     Deposit gas budget per L1 block. Zero closes deposits.
    /// @custom:field elasticityMultiplier Must be > 0 and must divide maxResourceLimit exactly. Use
    ///                                    1 alongside a zero limit.
    /// @custom:field systemTxMaxGas       Gas reserved for the L1 attributes deposit. Must satisfy
    ///                                    maxResourceLimit + systemTxMaxGas <= gasLimit.
    struct ResourceConfigOverride {
        bool enabled;
        uint32 maxResourceLimit;
        uint8 elasticityMultiplier;
        uint32 systemTxMaxGas;
    }

    struct DeployOPChainInput {
        // Roles
        address opChainProxyAdminOwner;
        address systemConfigOwner;
        address batcher;
        address unsafeBlockSigner;
        address proposer;
        address challenger;
        uint32 basefeeScalar;
        uint32 blobBaseFeeScalar;
        uint256 l2ChainId;
        address opcm;
        string saltMixer;
        uint64 gasLimit;
        // Configurable dispute game inputs
        GameType disputeGameType;
        Claim disputeAbsolutePrestate;
        Proposal startingAnchorRoot;
        Claim cannonAbsolutePrestate;
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
        // Optional override for the L1 resource config at SystemConfig initialization.
        ResourceConfigOverride resourceConfigOverride;
    }
}

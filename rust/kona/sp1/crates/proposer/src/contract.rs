//! Contract bindings for the super-root ZK dispute game.
//!
//! Hand-written subsets of the ABIs the proposer needs, rewritten for the
//! monorepo contracts from op-succinct's `fault-proof/src/contract.rs`
//! (@ 13716c2c). Sources of truth:
//! - `packages/contracts-bedrock/src/dispute/zk/ZKDisputeGame.sol`
//! - `packages/contracts-bedrock/interfaces/dispute/IDisputeGameFactory.sol`
//! - `packages/contracts-bedrock/src/dispute/AnchorStateRegistry.sol`
//! - `packages/contracts-bedrock/src/dispute/DelayedWETH.sol`

use alloy_primitives::U256;
use alloy_sol_types::sol;
use anyhow::{Error, anyhow};
use serde::{Deserialize, Serialize};

sol! {
    #[derive(Debug, PartialEq, Eq, Serialize, Deserialize)]
    #[sol(rpc)]
    contract DisputeGameFactory {
        function gameCount() external view returns (uint256 gameCount_);

        function gameAtIndex(uint256 _index)
            external
            view
            returns (uint32 gameType_, uint64 timestamp_, address proxy_);

        /// UUID-based duplicate probe: returns the existing proxy (or zero
        /// address) for exact (gameType, rootClaim, extraData) params.
        function games(uint32 _gameType, bytes32 _rootClaim, bytes calldata _extraData)
            external
            view
            returns (address proxy_, uint64 timestamp_);

        function create(uint32 _gameType, bytes32 _rootClaim, bytes calldata _extraData)
            external
            payable
            returns (address proxy_);

        function gameImpls(uint32 _gameType) external view returns (address impl_);

        function initBonds(uint32 _gameType) external view returns (uint256 bond_);

        /// Packed immutable game args registered for a game type
        /// (140 bytes for the ZK game, see `LibGameArgs.sol`).
        function gameArgs(uint32 _gameType) external view returns (bytes memory args_);

        event DisputeGameCreated(
            address indexed disputeProxy, uint32 indexed gameType, bytes32 indexed rootClaim
        );
    }

    /// Proposal lifecycle status, mirroring ZKDisputeGame.sol `ProposalStatus`.
    #[derive(Debug, PartialEq, Eq, Serialize, Deserialize)]
    enum ProposalStatus {
        Unchallenged,
        Challenged,
        UnchallengedAndValidProofProvided,
        ChallengedAndValidProofProvided,
        Resolved,
    }

    #[derive(Debug, PartialEq, Eq, Serialize, Deserialize)]
    #[sol(rpc)]
    contract ZKDisputeGame {
        /// Claim state. Field order matches ZKDisputeGame.sol `ClaimData`
        /// (NOT op-succinct's: `counteredBy` is renamed `challenger` and the
        /// field order differs).
        struct ClaimData {
            uint32 parentIndex;
            ProposalStatus status;
            address challenger;
            address prover;
            uint64 deadline;
            bytes32 claim;
        }

        function claimData()
            external
            view
            returns (
                uint32 parentIndex,
                ProposalStatus status,
                address challenger,
                address prover,
                uint64 deadline,
                bytes32 claim
            );

        function status() external view returns (uint8 status_);
        function gameType() external view returns (uint32 gameType_);
        function gameCreator() external view returns (address creator_);
        function rootClaim() external view returns (bytes32 rootClaim_);
        function l1Head() external view returns (bytes32 l1Head_);
        /// Super-root timestamp of the proposal (uint256-typed on-chain,
        /// always fits u64).
        function l2SequenceNumber() external view returns (uint256 l2SequenceNumber_);
        function startingSequenceNumber() external view returns (uint256 startingSequenceNumber_);
        function startingRootHash() external view returns (bytes32 startingRootHash_);
        function parentIndex() external view returns (uint32 parentIndex_);
        function absolutePrestate() external view returns (bytes32 absolutePrestate_);
        function verifier() external view returns (address verifier_);
        function weth() external view returns (address weth_);
        function anchorStateRegistry() external view returns (address registry_);
        function maxChallengeDuration() external view returns (uint64 maxChallengeDuration_);
        function maxProveDuration() external view returns (uint64 maxProveDuration_);
        function wasRespectedGameTypeWhenCreated() external view returns (bool wasRespected_);
        function gameOver() external view returns (bool gameOver_);
        function credit(address _recipient) external view returns (uint256 credit_);
        function resolvedAt() external view returns (uint64 resolvedAt_);

        function resolve() external returns (uint8 status_);
        function claimCredit(address _recipient) external;
    }

    #[derive(Debug, PartialEq, Eq, Serialize, Deserialize)]
    #[sol(rpc)]
    contract AnchorStateRegistry {
        function getAnchorRoot() external view returns (bytes32 root_, uint256 l2SequenceNumber_);
        function anchorGame() external view returns (address anchorGame_);
        function isGameFinalized(address _game) external view returns (bool isGameFinalized_);
        function respectedGameType() external view returns (uint32 gameType_);
    }

    #[derive(Debug, PartialEq, Eq, Serialize, Deserialize)]
    #[sol(rpc)]
    contract DelayedWETH {
        /// Withdrawal request created by ZKDisputeGame.claimCredit phase 1.
        function withdrawals(address _game, address _recipient)
            external
            view
            returns (uint256 amount_, uint256 timestamp_);

        /// Global withdrawal delay in seconds.
        function delay() external view returns (uint256 delay_);
    }
}

/// Game resolution status, mirroring
/// `packages/contracts-bedrock/src/dispute/lib/Types.sol` `GameStatus`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[repr(u8)]
pub enum GameStatus {
    IN_PROGRESS = 0,
    CHALLENGER_WINS = 1,
    DEFENDER_WINS = 2,
}

impl TryFrom<u8> for GameStatus {
    type Error = Error;

    fn try_from(value: u8) -> Result<Self, Self::Error> {
        match value {
            0 => Ok(Self::IN_PROGRESS),
            1 => Ok(Self::CHALLENGER_WINS),
            2 => Ok(Self::DEFENDER_WINS),
            _ => Err(anyhow!("invalid game status: {value}")),
        }
    }
}

/// Decoded ZK game args (140-byte packed layout from `LibGameArgs.sol`;
/// mirrors `op-challenger/game/fault/contracts/gameargs` `ZKGameArgs`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ZKGameArgs {
    pub absolute_prestate: alloy_primitives::B256,
    pub verifier: alloy_primitives::Address,
    pub max_challenge_duration: u64,
    pub max_prove_duration: u64,
    pub challenger_bond: U256,
    pub anchor_state_registry: alloy_primitives::Address,
    pub weth: alloy_primitives::Address,
}

/// Exact length of the packed ZK game args blob.
pub const ZK_GAME_ARGS_LENGTH: usize = 140;

impl ZKGameArgs {
    /// Decodes the packed layout: absolutePrestate(32) || verifier(20) ||
    /// maxChallengeDuration(8) || maxProveDuration(8) || challengerBond(32)
    /// || anchorStateRegistry(20) || weth(20).
    pub fn decode(args: &[u8]) -> Result<Self, Error> {
        if args.len() != ZK_GAME_ARGS_LENGTH {
            return Err(anyhow!("invalid ZK game args length: {}", args.len()));
        }
        Ok(Self {
            absolute_prestate: alloy_primitives::B256::from_slice(&args[0..32]),
            verifier: alloy_primitives::Address::from_slice(&args[32..52]),
            max_challenge_duration: u64::from_be_bytes(args[52..60].try_into().unwrap()),
            max_prove_duration: u64::from_be_bytes(args[60..68].try_into().unwrap()),
            challenger_bond: U256::from_be_slice(&args[68..100]),
            anchor_state_registry: alloy_primitives::Address::from_slice(&args[100..120]),
            weth: alloy_primitives::Address::from_slice(&args[120..140]),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{address, b256};

    /// Pins the enum numeric values to Types.sol / ZKDisputeGame.sol.
    #[test]
    fn game_status_values_match_types_sol() {
        assert_eq!(GameStatus::IN_PROGRESS as u8, 0);
        assert_eq!(GameStatus::CHALLENGER_WINS as u8, 1);
        assert_eq!(GameStatus::DEFENDER_WINS as u8, 2);
        assert!(GameStatus::try_from(3).is_err());
    }

    #[test]
    fn proposal_status_values_match_zk_dispute_game_sol() {
        assert_eq!(ProposalStatus::Unchallenged as u8, 0);
        assert_eq!(ProposalStatus::Challenged as u8, 1);
        assert_eq!(ProposalStatus::UnchallengedAndValidProofProvided as u8, 2);
        assert_eq!(ProposalStatus::ChallengedAndValidProofProvided as u8, 3);
        assert_eq!(ProposalStatus::Resolved as u8, 4);
    }

    /// Same test vectors as Go's `TestZKGameArgsPack`
    /// (op-challenger/game/fault/contracts/gameargs).
    #[test]
    fn zk_game_args_decode() {
        let prestate = b256!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
        let verifier = address!("1111111111111111111111111111111111111111");
        let asr = address!("2222222222222222222222222222222222222222");
        let weth = address!("3333333333333333333333333333333333333333");
        let bond = U256::from(10).pow(U256::from(18));

        let mut packed = Vec::with_capacity(ZK_GAME_ARGS_LENGTH);
        packed.extend_from_slice(prestate.as_slice());
        packed.extend_from_slice(verifier.as_slice());
        packed.extend_from_slice(&3600u64.to_be_bytes());
        packed.extend_from_slice(&7200u64.to_be_bytes());
        packed.extend_from_slice(&bond.to_be_bytes::<32>());
        packed.extend_from_slice(asr.as_slice());
        packed.extend_from_slice(weth.as_slice());

        let decoded = ZKGameArgs::decode(&packed).unwrap();
        assert_eq!(
            decoded,
            ZKGameArgs {
                absolute_prestate: prestate,
                verifier,
                max_challenge_duration: 3600,
                max_prove_duration: 7200,
                challenger_bond: bond,
                anchor_state_registry: asr,
                weth,
            }
        );

        assert!(ZKGameArgs::decode(&packed[..139]).is_err());
        assert!(ZKGameArgs::decode(&[]).is_err());
    }
}

#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/alloy.jpg",
    html_favicon_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/favicon.ico"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![no_std]

extern crate alloc;
use alloc::vec::Vec;
use alloy_hardforks::{EthereumHardfork, hardfork};
pub use alloy_hardforks::{EthereumHardforks, ForkCondition};
use alloy_primitives::U256;
use core::ops::Index;

hardfork!(
    /// The name of an optimism hardfork.
    ///
    /// When building a list of hardforks for a chain, it's still expected to zip with
    /// [`EthereumHardfork`].
    #[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
    #[derive(Default)]
    OpHardfork {
        /// Bedrock: <https://blog.oplabs.co/introducing-optimism-bedrock>.
        Bedrock,
        /// Regolith: <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/superchain-upgrades.md#regolith>.
        Regolith,
        /// <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/superchain-upgrades.md#canyon>.
        Canyon,
        /// Ecotone: <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/superchain-upgrades.md#ecotone>.
        Ecotone,
        /// Fjord: <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/superchain-upgrades.md#fjord>
        Fjord,
        /// Granite: <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/superchain-upgrades.md#granite>
        Granite,
        /// Holocene: <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/superchain-upgrades.md#holocene>
        Holocene,
        /// Isthmus: <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/isthmus/overview.md>
        Isthmus,
        /// Jovian: <https://github.com/ethereum-optimism/specs/tree/main/specs/protocol/jovian>
        #[default]
        Jovian,
        /// Karst: <https://github.com/ethereum-optimism/specs/tree/main/specs/protocol/karst>
        Karst,
        /// Lagoon: <https://specs.optimism.io/protocol/lagoon/overview.html>
        Lagoon,
    }
);

impl OpHardfork {
    /// Returns index of `self` in sorted canonical array.
    pub const fn idx(&self) -> usize {
        *self as usize
    }

    /// UPSTREAM-MIRROR(set): alloy-hardforks@0.4.11 `alloy_hardforks::EthereumHardfork`
    ///
    /// A new upstream variant needs an explicit decision here. The `VARIANTS` loop below
    /// only smoke-tests lookup; semantic mappings remain explicit assertions.
    ///
    /// Returns the L1 ([`EthereumHardfork`]) hardfork whose L2-relevant changes this OP
    /// hardfork newly activates, if any.
    ///
    /// This is the canonical OP fork → implied L1 fork table. Every other view of the
    /// relation ([`Self::implied_l1_fork`], [`Self::activating_op_fork`]) derives from it.
    pub const fn activates_l1_fork(self) -> Option<EthereumHardfork> {
        match self {
            Self::Bedrock => Some(EthereumHardfork::Paris),
            Self::Canyon => Some(EthereumHardfork::Shanghai),
            Self::Ecotone => Some(EthereumHardfork::Cancun),
            Self::Isthmus => Some(EthereumHardfork::Prague),
            Self::Karst => Some(EthereumHardfork::Osaka),
            _ => None,
        }
    }

    /// Returns the latest L1 ([`EthereumHardfork`]) hardfork whose changes are active on L2
    /// once this OP hardfork is active.
    ///
    /// This is the cumulative view of [`Self::activates_l1_fork`]: forks that don't activate
    /// an L1 fork themselves inherit the one implied by their predecessors.
    pub fn implied_l1_fork(self) -> EthereumHardfork {
        Self::VARIANTS[..=self.idx()]
            .iter()
            .rev()
            .find_map(|fork| fork.activates_l1_fork())
            .expect("Bedrock implies Paris, so every OP fork has an implied L1 fork")
    }

    /// Returns the OP hardfork that activates the given L1 ([`EthereumHardfork`]) hardfork's
    /// changes on L2, if any: the earliest OP fork whose [`Self::implied_l1_fork`] is at
    /// least the given fork. In particular, every L1 fork up to and including Paris is
    /// activated by Bedrock. L1 forks newer than the latest implied one (e.g. the
    /// blob-parameter-only forks) return `None`.
    ///
    /// Note that this ignores L2 chain history: pre-Bedrock L1 forks activated on OP chains
    /// at chain-specific blocks (or not at all, like Dao), which only the chain's own fork
    /// schedule can answer.
    pub fn activating_op_fork(l1_fork: EthereumHardfork) -> Option<Self> {
        Self::VARIANTS.iter().copied().find(|fork| fork.implied_l1_fork() >= l1_fork)
    }

    /// Returns every OP hardfork at or after `self`, in canonical (ascending) order.
    ///
    /// Rules that apply to "this fork and every later fork"
    /// can iterate this instead of hard-coding a fork list, covering future hardforks
    /// automatically.
    pub fn forks_from(self) -> impl Iterator<Item = Self> + Clone {
        Self::VARIANTS.iter().copied().filter(move |fork| fork.idx() >= self.idx())
    }
}

/// Extends [`EthereumHardforks`] with optimism helper methods.
#[auto_impl::auto_impl(&, Arc)]
pub trait OpHardforks: EthereumHardforks {
    /// Retrieves [`ForkCondition`] by an [`OpHardfork`]. If `fork` is not present, returns
    /// [`ForkCondition::Never`].
    fn op_fork_activation(&self, fork: OpHardfork) -> ForkCondition;

    /// Returns `true` if the block at `block_timestamp` is the activation block of some fork at
    /// or after Jovian, given its parent block's `parent_timestamp` — i.e. such a fork is active
    /// at this block but was not active at its parent. Since Jovian, fork-activation blocks must
    /// contain only deposit transactions — no user (non-deposit) txs.
    fn is_no_user_tx_activation_block(&self, parent_timestamp: u64, block_timestamp: u64) -> bool {
        OpHardfork::Jovian.forks_from().any(|fork| {
            let activation = self.op_fork_activation(fork);
            activation.active_at_timestamp(block_timestamp) &&
                !activation.active_at_timestamp(parent_timestamp)
        })
    }

    /// Convenience method to check if [`OpHardfork::Bedrock`] is active at a given block
    /// number.
    fn is_bedrock_active_at_block(&self, block_number: u64) -> bool {
        self.op_fork_activation(OpHardfork::Bedrock).active_at_block(block_number)
    }

    /// Returns `true` if [`Regolith`](OpHardfork::Regolith) is active at given block
    /// timestamp.
    fn is_regolith_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.op_fork_activation(OpHardfork::Regolith).active_at_timestamp(timestamp)
    }

    /// Returns `true` if [`Canyon`](OpHardfork::Canyon) is active at given block timestamp.
    fn is_canyon_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.op_fork_activation(OpHardfork::Canyon).active_at_timestamp(timestamp)
    }

    /// Returns `true` if [`Ecotone`](OpHardfork::Ecotone) is active at given block timestamp.
    fn is_ecotone_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.op_fork_activation(OpHardfork::Ecotone).active_at_timestamp(timestamp)
    }

    /// Returns `true` if [`Fjord`](OpHardfork::Fjord) is active at given block timestamp.
    fn is_fjord_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.op_fork_activation(OpHardfork::Fjord).active_at_timestamp(timestamp)
    }

    /// Returns `true` if [`Granite`](OpHardfork::Granite) is active at given block timestamp.
    fn is_granite_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.op_fork_activation(OpHardfork::Granite).active_at_timestamp(timestamp)
    }

    /// Returns `true` if [`Holocene`](OpHardfork::Holocene) is active at given block
    /// timestamp.
    fn is_holocene_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.op_fork_activation(OpHardfork::Holocene).active_at_timestamp(timestamp)
    }

    /// Returns `true` if [`Isthmus`](OpHardfork::Isthmus) is active at given block
    /// timestamp.
    fn is_isthmus_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.op_fork_activation(OpHardfork::Isthmus).active_at_timestamp(timestamp)
    }

    /// Returns `true` if [`Jovian`](OpHardfork::Jovian) is active at given block
    /// timestamp.
    fn is_jovian_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.op_fork_activation(OpHardfork::Jovian).active_at_timestamp(timestamp)
    }

    /// Returns `true` if [`Karst`](OpHardfork::Karst) is active at given block
    /// timestamp.
    fn is_karst_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.op_fork_activation(OpHardfork::Karst).active_at_timestamp(timestamp)
    }

    /// Returns `true` if [`Lagoon`](OpHardfork::Lagoon) is active at given block
    /// timestamp.
    fn is_lagoon_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.op_fork_activation(OpHardfork::Lagoon).active_at_timestamp(timestamp)
    }

    /// Returns `true` if the interop feature is active at the given block timestamp.
    ///
    /// Currently equivalent to [`Self::is_lagoon_active_at_timestamp`] because Lagoon
    /// is the hard fork that activates interop. Kept as a separate method so the
    /// feature toggle can diverge from the hard fork accessor if interop's activation
    /// is ever decoupled from Lagoon.
    fn is_interop_active_at_timestamp(&self, timestamp: u64) -> bool {
        self.is_lagoon_active_at_timestamp(timestamp)
    }
}

/// A type allowing to configure activation [`ForkCondition`]s for a given list of
/// [`OpHardfork`]s.
///
/// Zips together [`EthereumHardfork`]s and [`OpHardfork`]s. Optimism hard forks, at least,
/// whenever Ethereum hard forks. When Ethereum hard forks, a new [`OpHardfork`] piggybacks on top
/// of the new [`EthereumHardfork`] to include (or to noop) the L1 changes on L2.
///
/// Optimism can also hard fork independently of Ethereum. The relation between Ethereum and
/// Optimism hard forks is described by predicate [`EthereumHardfork`] `=>` [`OpHardfork`], since
/// an OP chain can undergo an [`OpHardfork`] without an [`EthereumHardfork`], but not the other
/// way around.
#[derive(Debug, Clone)]
pub struct OpChainHardforks {
    /// Ordered list of OP hardfork activations.
    forks: Vec<(OpHardfork, ForkCondition)>,
}

impl OpChainHardforks {
    /// Creates a new [`OpChainHardforks`] with the given list of forks. The input list is sorted
    /// w.r.t. the hardcoded canonicity of [`OpHardfork`]s.
    pub fn new(forks: impl IntoIterator<Item = (OpHardfork, ForkCondition)>) -> Self {
        let mut forks = forks.into_iter().collect::<Vec<_>>();
        forks.sort();
        Self { forks }
    }
}

impl EthereumHardforks for OpChainHardforks {
    fn ethereum_fork_activation(&self, fork: EthereumHardfork) -> ForkCondition {
        if self.forks.is_empty() {
            return ForkCondition::Never;
        }

        // The L1 fork never activates if the OP fork activating it is out of bounds.
        if let Some(op_fork) = OpHardfork::activating_op_fork(fork) &&
            self.forks.len() <= op_fork.idx()
        {
            return ForkCondition::Never;
        }

        self[fork]
    }
}

impl OpHardforks for OpChainHardforks {
    fn op_fork_activation(&self, fork: OpHardfork) -> ForkCondition {
        // check index out of bounds
        if self.forks.len() <= fork.idx() {
            return ForkCondition::Never;
        }
        self[fork]
    }
}

impl Index<OpHardfork> for OpChainHardforks {
    type Output = ForkCondition;

    fn index(&self, hf: OpHardfork) -> &Self::Output {
        use OpHardfork::{
            Bedrock, Canyon, Ecotone, Fjord, Granite, Holocene, Isthmus, Jovian, Karst, Lagoon,
            Regolith,
        };

        match hf {
            Bedrock => &self.forks[Bedrock.idx()].1,
            Regolith => &self.forks[Regolith.idx()].1,
            Canyon => &self.forks[Canyon.idx()].1,
            Ecotone => &self.forks[Ecotone.idx()].1,
            Fjord => &self.forks[Fjord.idx()].1,
            Granite => &self.forks[Granite.idx()].1,
            Holocene => &self.forks[Holocene.idx()].1,
            Isthmus => &self.forks[Isthmus.idx()].1,
            Jovian => &self.forks[Jovian.idx()].1,
            Karst => &self.forks[Karst.idx()].1,
            Lagoon => &self.forks[Lagoon.idx()].1,
        }
    }
}

impl Index<EthereumHardfork> for OpChainHardforks {
    type Output = ForkCondition;

    fn index(&self, hf: EthereumHardfork) -> &Self::Output {
        use EthereumHardfork::{
            Berlin, Byzantium, Constantinople, Dao, Frontier, Homestead, Istanbul, MuirGlacier,
            Paris, Petersburg, SpuriousDragon, Tangerine,
        };

        match hf {
            // Dao Hardfork is not needed for OpChainHardforks
            Dao => &ForkCondition::Never,
            // Pre-Bedrock L1 history (e.g. OP Mainnet's legacy Berlin block) isn't modelled.
            Frontier | Homestead | Tangerine | SpuriousDragon | Byzantium | Constantinople |
            Petersburg | Istanbul | MuirGlacier | Berlin => &ForkCondition::ZERO_BLOCK,
            Paris => &ForkCondition::TTD {
                activation_block_number: 0,
                fork_block: Some(0),
                total_difficulty: U256::ZERO,
            },
            // Everything else activates with the OP fork that implies it (Bedrock for
            // London through GrayGlacier); L1 forks without an L2 equivalent (the BPO
            // forks, Amsterdam, ...) never activate.
            _ => OpHardfork::activating_op_fork(hf)
                .map_or(&ForkCondition::Never, |op_fork| &self[op_fork]),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use core::str::FromStr;

    extern crate alloc;

    #[test]
    fn forks_from_returns_self_and_later_forks_in_order() {
        let from_jovian: alloc::vec::Vec<_> = OpHardfork::Jovian.forks_from().collect();
        assert_eq!(from_jovian, [OpHardfork::Jovian, OpHardfork::Karst, OpHardfork::Lagoon]);

        // `self` is included; everything before it is excluded.
        assert_eq!(
            OpHardfork::Lagoon.forks_from().collect::<alloc::vec::Vec<_>>(),
            [OpHardfork::Lagoon]
        );
        assert!(OpHardfork::Bedrock.forks_from().eq(OpHardfork::VARIANTS.iter().copied()));

        // Ascending and contiguous: every returned fork has idx >= the anchor's.
        assert!(from_jovian.iter().all(|f| f.idx() >= OpHardfork::Jovian.idx()));
    }

    /// Bedrock at genesis, then each fork through `last` at `100 * idx`; later forks are absent.
    fn staggered_hardforks(last: OpHardfork) -> OpChainHardforks {
        OpChainHardforks::new(OpHardfork::VARIANTS[..=last.idx()].iter().map(|&fork| {
            let condition = if fork == OpHardfork::Bedrock {
                ForkCondition::Block(0)
            } else {
                ForkCondition::Timestamp(100 * fork.idx() as u64)
            };
            (fork, condition)
        }))
    }

    #[test]
    fn is_no_user_tx_activation_block_detects_first_fork_block() {
        let forks = staggered_hardforks(OpHardfork::Karst);
        let jovian = 100 * OpHardfork::Jovian.idx() as u64;

        // First block at/after the Jovian timestamp is the activation block.
        assert!(forks.is_no_user_tx_activation_block(jovian - 2, jovian));
        assert!(forks.is_no_user_tx_activation_block(jovian - 2, jovian + 2));
        // Already active at the parent, or not yet active at the block.
        assert!(!forks.is_no_user_tx_activation_block(jovian, jovian + 2));
        assert!(!forks.is_no_user_tx_activation_block(jovian - 4, jovian - 2));
        // Pre-Jovian fork activations don't count.
        let isthmus = 100 * OpHardfork::Isthmus.idx() as u64;
        assert!(!forks.is_no_user_tx_activation_block(isthmus - 2, isthmus));
    }

    #[test]
    fn check_op_hardfork_from_str() {
        let hardfork_str = [
            "beDrOck", "rEgOlITH", "cAnYoN", "eCoToNe", "FJorD", "GRaNiTe", "hOlOcEnE", "isthMUS",
            "jOvIaN", "kArSt", "laGoON",
        ];
        let expected_hardforks = [
            OpHardfork::Bedrock,
            OpHardfork::Regolith,
            OpHardfork::Canyon,
            OpHardfork::Ecotone,
            OpHardfork::Fjord,
            OpHardfork::Granite,
            OpHardfork::Holocene,
            OpHardfork::Isthmus,
            OpHardfork::Jovian,
            OpHardfork::Karst,
            OpHardfork::Lagoon,
        ];

        let hardforks: alloc::vec::Vec<OpHardfork> =
            hardfork_str.iter().map(|h| OpHardfork::from_str(h).unwrap()).collect();

        assert_eq!(hardforks, expected_hardforks);
    }

    #[test]
    fn check_nonexistent_hardfork_from_str() {
        assert!(OpHardfork::from_str("not a hardfork").is_err());
    }

    #[test]
    fn fork_conditions_follow_schedule() {
        let forks = staggered_hardforks(OpHardfork::Karst);
        assert_eq!(forks[OpHardfork::Bedrock], ForkCondition::Block(0));
        for fork in OpHardfork::Regolith.forks_from().take_while(|f| *f <= OpHardfork::Karst) {
            let activation = 100 * fork.idx() as u64;
            assert_eq!(forks[fork], ForkCondition::Timestamp(activation));
            assert!(forks.op_fork_activation(fork).active_at_timestamp(activation));
            assert!(!forks.op_fork_activation(fork).active_at_timestamp(activation - 1));
        }
        // Forks past the end of the list never activate.
        assert_eq!(forks.op_fork_activation(OpHardfork::Lagoon), ForkCondition::Never);
        assert!(!forks.is_lagoon_active_at_timestamp(u64::MAX));
    }

    #[test]
    fn implied_l1_fork_table() {
        use EthereumHardfork::{Cancun, Osaka, Paris, Prague, Shanghai};

        let expected = [
            (OpHardfork::Bedrock, Paris),
            (OpHardfork::Regolith, Paris),
            (OpHardfork::Canyon, Shanghai),
            (OpHardfork::Ecotone, Cancun),
            (OpHardfork::Fjord, Cancun),
            (OpHardfork::Granite, Cancun),
            (OpHardfork::Holocene, Cancun),
            (OpHardfork::Isthmus, Prague),
            (OpHardfork::Jovian, Prague),
            (OpHardfork::Karst, Osaka),
            (OpHardfork::Lagoon, Osaka),
        ];
        assert_eq!(expected.len(), OpHardfork::VARIANTS.len());
        for (op_fork, l1_fork) in expected {
            assert_eq!(op_fork.implied_l1_fork(), l1_fork, "{op_fork}");
        }
    }

    #[test]
    fn implied_l1_fork_is_monotonic() {
        for window in OpHardfork::VARIANTS.windows(2) {
            assert!(
                window[0].implied_l1_fork() <= window[1].implied_l1_fork(),
                "implied L1 fork must not decrease from {} to {}",
                window[0],
                window[1]
            );
        }
    }

    #[test]
    fn activating_op_fork_inverts_activates_l1_fork() {
        for op_fork in OpHardfork::VARIANTS.iter().copied() {
            if let Some(l1_fork) = op_fork.activates_l1_fork() {
                assert_eq!(OpHardfork::activating_op_fork(l1_fork), Some(op_fork));
            }
        }
        // Bedrock activates the whole pre-merge range up to and including Paris.
        assert_eq!(
            OpHardfork::activating_op_fork(EthereumHardfork::London),
            Some(OpHardfork::Bedrock)
        );
        assert_eq!(
            OpHardfork::activating_op_fork(EthereumHardfork::GrayGlacier),
            Some(OpHardfork::Bedrock)
        );
        // L1 forks newer than the latest implied one are not activated by any OP fork.
        assert_eq!(OpHardfork::activating_op_fork(EthereumHardfork::Bpo1), None);
        assert_eq!(OpHardfork::activating_op_fork(EthereumHardfork::Amsterdam), None);
        assert_eq!(OpHardfork::activating_op_fork(EthereumHardfork::Bogota), None);
    }

    #[test]
    fn ethereum_fork_activation_follows_op_fork_activation() {
        use EthereumHardfork::{
            ArrowGlacier, Cancun, GrayGlacier, London, Osaka, Prague, Shanghai,
        };
        use OpHardfork::{Bedrock, Canyon, Ecotone, Isthmus, Karst};

        let forks = staggered_hardforks(OpHardfork::Karst);
        for (l1_fork, op_fork) in [
            (London, Bedrock),
            (ArrowGlacier, Bedrock),
            (GrayGlacier, Bedrock),
            (Shanghai, Canyon),
            (Cancun, Ecotone),
            (Prague, Isthmus),
            (Osaka, Karst),
        ] {
            assert_eq!(
                forks.ethereum_fork_activation(l1_fork),
                forks.op_fork_activation(op_fork),
                "{l1_fork} must activate with {op_fork}"
            );
        }
        // Paris is absent above: although Bedrock activates it, it doesn't share Bedrock's
        // ForkCondition — Paris activates via a TTD condition while Bedrock activates by block
        // number.
        assert!(matches!(
            forks.ethereum_fork_activation(EthereumHardfork::Paris),
            ForkCondition::TTD { .. }
        ));
        // L1 forks without an L2 equivalent never activate.
        assert_eq!(forks.ethereum_fork_activation(EthereumHardfork::Bpo1), ForkCondition::Never);
    }

    // https://github.com/alloy-rs/hardforks/issues/63
    #[test]
    fn test_ethereum_fork_activation_consistency() {
        let forks = staggered_hardforks(OpHardfork::Karst);
        for ethereum_hardfork in EthereumHardfork::VARIANTS {
            let _ = forks.ethereum_fork_activation(*ethereum_hardfork);
        }
        for op_hardfork in OpHardfork::VARIANTS {
            let _ = forks.op_fork_activation(*op_hardfork);
        }
    }
}

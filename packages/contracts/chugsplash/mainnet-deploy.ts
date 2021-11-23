import { ChugSplashConfig } from '@eth-optimism/chugsplash'

const config: ChugSplashConfig = {
  options: {
    name: 'Optimistic Ethereum (mainnet)',
    owner: '0x4cdC4f412355F296C2cf261210Cc9274404E442b',
  },
  contracts: {
    ChainStorageContainer_CTC_batches: {
      source: 'ChainStorageContainer',
      variables: {
        owner: '{{ contracts.CanonicalTransactionChain }}',
      },
    },
    ChainStorageContainer_SCC_batches: {
      source: 'ChainStorageContainer',
      variables: {
        owner: '{{ contracts.StateCommitmentChain }}',
      },
    },
    CanonicalTransactionChain: {
      source: 'CanonicalTransactionChain',
      variables: {
        enqueueGasCost: 60_000,
        l2GasDiscountDivisor: 32,
        enqueueL2GasPrepaid: 1_920_000,
        maxTransactionGasLimit: 15_000_000,
        batches: '{{ contracts.ChainStorageContainer_CTC_batches }}',
        sequencer: '0x6887246668a3b87F54DeB3b94Ba47a6f63F32985',
      },
    },
    StateCommitmentChain: {
      source: 'StateCommitmentChain',
      variables: {
        FRAUD_PROOF_WINDOW: 604_800,
        SEQUENCER_PUBLISH_WINDOW: 12_592_000,
        batches: '{{ contracts.ChainStorageContainer_SCC_batches }}',
        bondManager: '{{ contracts.BondManager }}',
        ctc: '{{ contracts.CanonicalTransactionChain }}',
        fraudVerifier: '0x0000000000000000000000000000000000000000',
        proposer: '0x473300df21D047806A082244b417f96b32f13A33',
      },
    },
    BondManager: {
      source: 'BondManager',
      variables: {
        proposer: '0x473300df21D047806A082244b417f96b32f13A33',
      },
    },
    L1CrossDomainMessenger: {
      source: 'L1CrossDomainMessenger',
      variables: {
        xDomainMsgSender: '0x000000000000000000000000000000000000dEaD',
        _initialized: true,
        _owner: '0x4cdC4f412355F296C2cf261210Cc9274404E442b',
        _paused: false,
        _status: 1, // _NOT_ENTERED
      },
    },
    L1StandardBridge: {
      source: 'L1StandardBridge',
      variables: {
        l2TokenBridge: '0x4200000000000000000000000000000000000010',
        messenger: '{{ contracts.L1CrossDomainMessenger }}',
      },
    },
  },
}

export default config

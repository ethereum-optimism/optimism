import { UserChugSplashConfig } from '@chugsplash/core'
import { predeploys } from '../src/constants'

const config: UserChugSplashConfig = {
  options: {
    projectName: 'optimism6',
  },
  contracts: {
    Proxy__OVM_L1CrossDomainMessenger: {
      contract: 'L1CrossDomainMessenger',
      variables: {
        _initialized: 1,
        _initializing: false,
        _owner: '0x68108902De3A5031197a6eB3b74b3b033e8E8e4d',
        _status: 1,
        _paused: false,
        __gap: [],
        spacer_0_0_20: '0x0000000000000000000000000000000000000000',
        spacer_201_0_32: {},
        spacer_202_0_32: {},
        OTHER_MESSENGER: predeploys.L2CrossDomainMessenger,
        xDomainMsgSender: '0x000000000000000000000000000000000000dEaD',
        msgNonce: 0,
        PORTAL: "{{ OptimismPortalProxy }}",
        successfulMessages: {},
        failedMessages: {},
        MAJOR_VERSION: 0,
        MINOR_VERSION: 0,
        PATCH_VERSION: 1,
      }
    },
    L1ERC721BridgeProxy: {
      contract: 'L1ERC721Bridge',
      variables: {
        __gap: [],
        deposits: {},
        MESSENGER: "{{ Proxy__OVM_L1CrossDomainMessenger }}",
        OTHER_BRIDGE: predeploys.L2ERC721Bridge,
        MAJOR_VERSION: 0,
        MINOR_VERSION: 0,
        PATCH_VERSION: 1,
      }
    },
    Proxy__OVM_L1StandardBridge: {
      contract: 'L1StandardBridge',
      variables: {
        spacer_0_0_20: '0x0000000000000000000000000000000000000000',
        spacer_1_0_20: '0x0000000000000000000000000000000000000000',
        deposits: {},
        __gap: [],
        MESSENGER: "{{ Proxy__OVM_L1CrossDomainMessenger }}",
        OTHER_BRIDGE: predeploys.L2StandardBridge,
        MAJOR_VERSION: 0,
        MINOR_VERSION: 0,
        PATCH_VERSION: 1,
      }
    },
    L2OutputOracleProxy: {
      contract: 'L2OutputOracle',
      variables: {
        _initialized: 1,
        _initializing: false,
        l2Outputs: [],
        SUBMISSION_INTERVAL: 20,
        startingBlockNumber: 0,
        startingTimestamp: 0,
        L2_BLOCK_TIME: 2,
        PROPOSER: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
        CHALLENGER: '0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65',
        MAJOR_VERSION: 0,
        MINOR_VERSION: 0,
        PATCH_VERSION: 1,
      }
    },
    OptimismPortalProxy: {
      contract: 'OptimismPortal',
      variables: {
        _initialized: 1,
        _initializing: false,
        __gap: [],
        provenWithdrawals: {},
        finalizedWithdrawals: {},
        params: {
          prevBaseFee: 1_000_000_000,
          prevBoughtGas: 0,
          prevBlockNum: 0,
        },
        FINALIZATION_PERIOD_SECONDS: 2,
        L2_ORACLE: "{{ L2OutputOracleProxy }}",
        l2Sender: '0x000000000000000000000000000000000000dEaD',
        MAJOR_VERSION: 0,
        MINOR_VERSION: 0,
        PATCH_VERSION: 1,
      }
    },
    SystemConfigProxy: {
      contract: 'SystemConfig',
      variables: {
        _initialized: 1,
        _initializing: false,
        __gap: [],
        _owner: '0xa0Ee7A142d267C1f36714E4a8F75612F20a79720',
        overhead: 0,
        scalar: 1,
        batcherHash: '0x0000000000000000000000003C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
        gasLimit: 15000000,
        MAJOR_VERSION: 0,
        MINOR_VERSION: 0,
        PATCH_VERSION: 1,
      }
    }
  }
}

export default config

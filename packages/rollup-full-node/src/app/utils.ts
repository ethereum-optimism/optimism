/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'

import * as waffle from 'ethereum-waffle'
import { FullnodeHandler, L2ToL1MessageSubmitter } from '../types'
import { Web3Provider } from 'ethers/providers'
import { providers } from 'ethers'
import { NoOpL2ToL1MessageSubmitter } from './message-submitter'
import { DefaultWeb3Handler } from './web3-rpc-handler'
import { FullnodeRpcServer } from './fullnode-rpc-server'

const log = getLogger('utils')

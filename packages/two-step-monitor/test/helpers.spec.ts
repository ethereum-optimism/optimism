import hre from 'hardhat'
import { Contract } from 'ethers'
import { toRpcHexString } from '@eth-optimism/core-utils'
import {
  getContractFactory,
  getContractInterface,
} from '@eth-optimism/contracts'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import { smock, FakeContract } from '@defi-wonderland/smock'

import { expect } from './setup'
import {
  findEventForStateBatch,
  findFirstUnfinalizedStateBatchIndex,
} from '../src'

describe('helpers', () => {
})

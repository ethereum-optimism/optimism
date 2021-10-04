import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'
import { ethers } from 'ethers'
import linker from 'solc/linker'
import { add0x } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { expect } from '../setup'
import { handlers } from '../../scripts/handlers'
import {
  Account,
  AccountType,
  SurgeryDataSources,
  EtherscanContract,
  ImmutableReference,
} from '../../scripts/types'
import etherscanData = require('../etherscan-contracts.json')
import { downloadAllSolcVersions } from '../../scripts/download-solc'
import { solcInput, getSolc, getMainContract } from '../../scripts/utils'
import { COMPILER_VERSIONS_TO_SOLC } from '../../scripts/constants'

const newSurgeryDataSources = (): SurgeryDataSources => {
  return {
    dump: [],
    genesis: [],
    pools: [],
    etherscanDump: etherscanData as EtherscanContract[],
    l1TestnetProvider: new ethers.providers.JsonRpcProvider(),
    l1TestnetWallet: ethers.Wallet.createRandom(),
    l1MainnetProvider: new ethers.providers.JsonRpcProvider(),
    l2Provider: new ethers.providers.JsonRpcProvider(),
  }
}

describe('Handlers', () => {
  before(async () => {
    await downloadAllSolcVersions()
  })

  const dummyAccount: Account = {
    address: '0x0000000000000000000000000000000000000420',
    nonce: 69,
    balance: '0',
    codeHash: '420e69',
    root: '420e69',
    code: '608060405',
    storage: {
      '0x0000000000000000000000000000000000000420':
        '0000000000000000000000000000000000000420',
    },
  }

  describe('EOA', () => {
    it('returns the account without code', async () => {
      const output = await handlers[AccountType.EOA](dummyAccount, null)
      expect(output.address).to.eq(dummyAccount.address)
      expect(output.nonce).to.eq(dummyAccount.nonce)
      expect(output.balance).to.eq(dummyAccount.balance)
      expect(output.codeHash).to.eq(KECCAK256_NULL_S)
      expect(output.root).to.eq(KECCAK256_RLP_S)
      expect(output.code).to.be.undefined
      expect(output.storage).to.be.undefined
    })
  })

  describe('Verified', () => {
    const handler = handlers[AccountType.VERIFIED]

    it('should be a function', async () => {
      expect(typeof handler).to.eq('function')
    })

    it('should compile a contract', async () => {
      const etherscanAccount = etherscanData[2]

      const account: Account = {
        address: etherscanAccount.contractAddress,
        nonce: 0,
        balance: '0',
        codeHash: etherscanAccount.hash,
        root: ethers.utils.keccak256(add0x(etherscanAccount.code)),
        code: etherscanAccount.code,
      }

      const dataSources = newSurgeryDataSources()
      const output = await handler(account, dataSources)

      // Address should not change
      expect(output.address).to.eq(account.address)
      // Nonce should not change
      expect(output.nonce).to.eq(account.nonce)
      // Balance should not change
      expect(output.balance).to.eq(account.balance)
      // Code hash should be different
      expect(output.codeHash).to.not.eq(account.codeHash)
      // Code should be different
      expect(output.code).to.not.eq(account.code)
      // TODO: deploy the contract and make sure it executes
    })

    // 75 has immutables
    it('should compile a contract with immutables', async () => {
      const dataSources = newSurgeryDataSources()
      const etherscanAccount = etherscanData[75]

      // Compile the contract
      const input = solcInput(etherscanAccount)
      const version =
        COMPILER_VERSIONS_TO_SOLC[etherscanAccount.compilerVersion]
      const currSolc = getSolc(version)
      const output = JSON.parse(currSolc.compile(JSON.stringify(input)))
      const contract = getMainContract(etherscanAccount, output)

      const immutableRefs: ImmutableReference =
        contract.evm.deployedBytecode.immutableReferences
      expect(Object.keys(immutableRefs).length).to.be.gte(1)

      // Compile using the handler
      const account: Account = {
        address: etherscanAccount.contractAddress,
        nonce: 0,
        balance: '0',
        codeHash: etherscanAccount.hash,
        root: ethers.utils.keccak256(add0x(etherscanAccount.code)),
        code: etherscanAccount.code,
      }
      const result = await handler(account, dataSources)
    })

    // 161 has a bunch of library references
    it('should compile a contract with libraries', async () => {
      const dataSources = newSurgeryDataSources()
      const etherscanAccount = etherscanData[161]

      // Compile the contract
      const input = solcInput(etherscanAccount)
      const version =
        COMPILER_VERSIONS_TO_SOLC[etherscanAccount.compilerVersion]
      const currSolc = getSolc(version)
      const output = JSON.parse(currSolc.compile(JSON.stringify(input)))
      const contract = getMainContract(etherscanAccount, output)
      // Search for library references
      const libs = linker.findLinkReferences(
        contract.evm.deployedBytecode.object
      )
      // There is one library
      expect(Object.keys(libs).length).to.be.gte(1)
      const lib = Object.keys(libs)[0]
      expect(contract.evm.deployedBytecode.object.includes(lib)).to.be.true

      const account: Account = {
        address: etherscanAccount.contractAddress,
        nonce: 0,
        balance: '0',
        codeHash: etherscanAccount.hash,
        root: ethers.utils.keccak256(add0x(etherscanAccount.code)),
        code: etherscanAccount.code,
      }
      const result = await handler(account, dataSources)

      expect(result.code.includes(lib)).to.be.false
    })
  })
})

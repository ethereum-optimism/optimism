import '../setup'
/* External Imports */
import { getLogger, remove0x, bufToHexString } from '@eth-optimism/core-utils'
import { DB, newInMemoryDB } from '@eth-optimism/core-db/'
import {
  createMockProvider, getWallets, deployContract
} from '@eth-optimism/rollup-full-node'
import { ethers } from 'ethers'
import { solidity } from 'ethereum-waffle'
import {
  Address,
  bytecodeToBuffer,
  EVMBytecode,
  Opcode,
  formatBytecode,
  bufferToBytecode,
  EVMOpcodeAndBytes,
  EVMOpcode,
  getPCOfEVMBytecodeIndex,
} from '@eth-optimism/rollup-core'

/* Internal Imports */

import * as SimpleStorage from '../contracts/build/SimpleStorage.json'
import * as SelfAware from '../contracts/build/SelfAware.json'
import * as CallerGetter from '../contracts/build/CallerGetter.json'
import * as CallerReturner from '../contracts/build/CallerReturner.json'
import * as TimeGetter from '../contracts/build/TimeGetter.json'

const log = getLogger('transpiler-full-node-integration')

describe.only(`Various opcodes should be usable in combination with transpiler and full node`, () => {
    // let fullnodeHandler: FullnodeHandler
    // let fullnodeRpcServer: FullnodeRpcServer
    // let baseUrl: string
  
    // let executionManagerAddress
    // let httpProvider
    // const host = '0.0.0.0'
    // const port = 9999   
  
    // before(async () => {
    //   fullnodeHandler = await DefaultWeb3Handler.create()
    //   fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)
  
    //   fullnodeRpcServer.listen()
  
    //   baseUrl = `http://${host}:${port}`
    //   log.debug(
    //     `setting up fullnode provider and get execution manager address...`
    //   )
    //   httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
    //   executionManagerAddress = await httpProvider.send(
    //     'ovm_getExecutionManagerAddress',
    //     []
    //   )
    //   log.debug(`execution manager address acquired: ${executionManagerAddress}`)
    // })
  
    // afterEach(() => {
    //   if (!!fullnodeRpcServer) {
    //     fullnodeRpcServer.close()
    //   }
    // })

    it('first one', async () => {
        log.debug(FullnodeRpcServer)
    })
})


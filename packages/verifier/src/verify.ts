import BigNum from 'bn.js'
import EthereumTx from 'ethereumjs-tx'

import { ABIObject, ABIResult, decodeResponse, encodeMethod } from './abi'
import { ACCOUNT, PREDICATE_ABI } from './constants'
import { VM } from './vm'

const vm = new VM({ enableHomestead: true, activatePrecompiles: true })
let nonce = new BigNum(0)

const initGenesis = async (): Promise<void> => {
  const genesisData = {
    [ACCOUNT.address]:
      '1606938044258990275541962092341162602522202993782792835301376',
  }
  await vm.generateGenesis(genesisData)
}

const createContract = async (bytecode: string): Promise<string> => {
  const contractCreationTx = new EthereumTx({
    data: bytecode,
    from: ACCOUNT.address,
    gasLimit: '0xffffffff',
    gasPrice: '0x01',
    nonce: '0x' + nonce.toString('hex'),
    value: '0x00',
  })
  contractCreationTx.sign(ACCOUNT.privateKey)
  const contractCreationTxResult = await vm.runTx({
    skipBalance: true,
    skipNonce: true,
    tx: contractCreationTx,
  })

  if (contractCreationTxResult.createdAddress === undefined) {
    throw new Error('Could not create contract.')
  }

  nonce = nonce.addn(1)
  return '0x' + contractCreationTxResult.createdAddress.toString('hex')
}

const getContractMethod = (name: string): ABIObject => {
  const method = PREDICATE_ABI.find((item) => {
    return item.name === name
  })

  if (method === undefined) {
    throw new Error('Could not find method name.')
  }

  return method
}

const callContractMethod = async (
  address: string,
  method: string,
  inputs: string[]
): Promise<ABIResult> => {
  const methodAbi = getContractMethod(method)
  const methodData = encodeMethod(methodAbi, inputs)
  const validationCallTx = new EthereumTx({
    data: methodData,
    gasLimit: '0xffffffff',
    gasPrice: '0x01',
    nonce: '0x01',
    to: address,
    value: '0x00',
  })
  validationCallTx.sign(ACCOUNT.privateKey)
  const result = await vm.runTx({
    skipBalance: true,
    skipNonce: true,
    tx: validationCallTx,
  })

  if (result.vm.exception === 0) {
    const error = result.vm.exceptionError
    throw new Error(`${error.errorType}: ${error.error}`)
  }

  nonce = nonce.addn(1)
  const decoded = decodeResponse(methodAbi, result.vm.return)
  return decoded
}

export const validStateTransition = async (
  oldState: string,
  newState: string,
  witness: string,
  bytecode: string
): Promise<boolean> => {
  await initGenesis()
  const contractAddress = await createContract(bytecode)
  const result = await callContractMethod(
    contractAddress,
    'validStateTransition',
    [oldState, newState, witness]
  )
  return result[0] as boolean
}

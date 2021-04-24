/* External Imports */
import { providers, BigNumber } from 'ethers'
import {
  BlockWithTransactions,
  TransactionResponse,
} from '@ethersproject/abstract-provider'
import { L2Transaction, L2Block, RollupInfo } from '@eth-optimism/core-utils'

/**
 * Unformatted Transaction & Blocks. This exists because Geth currently
 * does not return the correct fields & so this code renames those
 * poorly named fields
 */
interface UnformattedL2Transaction extends TransactionResponse {
  l1BlockNumber: string
  l1MessageSender: string
  signatureHashType: string
  queueOrigin: string
  rawTransaction: string
}

interface UnformattedL2Block extends BlockWithTransactions {
  stateRoot: string
  transactions: [UnformattedL2Transaction]
}

export class MockchainProvider extends providers.JsonRpcProvider {
  public mockBlockNumber: number = 1
  public numBlocksToReturn: number = 2
  public mockBlocks: L2Block[] = []
  public ctcAddr: string
  public sccAddr: string

  constructor(ctcAddr: string, sccAddr: string) {
    super('https://optimism.io')
    for (const block of BLOCKS) {
      if (block.number === 0) {
        // No need to convert genesis to an L2Block because it has no txs
        this.mockBlocks.push(block)
        continue
      }
      this.mockBlocks.push(this._toL2Block(block))
      this.ctcAddr = ctcAddr
      this.sccAddr = sccAddr
    }
  }

  public async getBlockNumber(): Promise<number> {
    // Increment our mock block number every time
    if (
      this.mockBlockNumber + this.numBlocksToReturn <
      this.mockBlocks.length
    ) {
      this.mockBlockNumber += this.numBlocksToReturn
    } else {
      return this.mockBlocks.length - 1
    }
    return this.mockBlockNumber
  }

  public async send(endpoint: string, params: []): Promise<any> {
    switch (endpoint) {
      case 'eth_chainId':
        return this.chainId()
      case 'rollup_getInfo':
        const info: RollupInfo = {
          mode: 'sequencer',
          syncing: false,
          ethContext: {
            timestamp: 0,
            blockNumber: 0,
          },
          rollupContext: {
            index: 0,
            queueIndex: 0,
          },
        }
        return info
      case 'eth_getBlockByNumber':
        if (params.length === 0) {
          throw new Error(`Invalid params for ${endpoint}`)
        }
        const blockNumber = BigNumber.from((params as any)[0]).toNumber()
        return this.mockBlocks[blockNumber]
      default:
        throw new Error('Unsupported endpoint!')
    }
  }

  public setNumBlocksToReturn(numBlocks: number): void {
    this.numBlocksToReturn = numBlocks
  }

  public setL2BlockData(
    tx: L2Transaction,
    timestamp?: number,
    stateRoot?: string,
    start: number = 1,
    end: number = this.mockBlocks.length
  ) {
    for (let i = start; i < end; i++) {
      this.mockBlocks[i].timestamp = timestamp
        ? timestamp
        : this.mockBlocks[i].timestamp
      this.mockBlocks[i].transactions[0] = {
        ...this.mockBlocks[i].transactions[0],
        ...tx,
      }
      this.mockBlocks[i].stateRoot = stateRoot
    }
  }

  public async getBlockWithTransactions(blockNumber: number): Promise<L2Block> {
    return this.mockBlocks[blockNumber]
  }

  public chainId(): number {
    // We know that mockBlocks will always have at least 1 value
    return this.mockBlocks[1].transactions[0].chainId
  }

  private _toL2Block(block: UnformattedL2Block): L2Block {
    const txType: number = parseInt(block.transactions[0].signatureHashType, 10)
    const l1BlockNumber: number = parseInt(
      block.transactions[0].l1BlockNumber,
      10
    )
    const queueOrigin: string = block.transactions[0].queueOrigin
    const l1TxOrigin: string = block.transactions[0].l1MessageSender
    const l2Transaction: L2Transaction = {
      ...block.transactions[0],
      // Rename the incorrectly named fields
      l1TxOrigin,
      queueOrigin,
      l1BlockNumber,
    }
    // Add an interface here to fix the type casing into L2Block during Object.assign
    interface PartialL2Block {
      transactions: [L2Transaction]
    }
    const partialBlock: PartialL2Block = {
      transactions: [l2Transaction],
    }
    const l2Block: L2Block = { ...block, ...partialBlock }
    return l2Block
  }
}

const BLOCKS = JSON.parse(`
[
    {
       "hash":"0xbc27fdbd1fee6e001438709ef57210bb7b2b1b8c23b65acb2d79161f4dc3cf05",
       "parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
       "number":0,
       "timestamp":1603651804,
       "nonce":"0x0000000000000042",
       "difficulty":1,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0x0000000000000000000000000000000000000000",
       "extraData":"0x1234",
       "transactions":[
       ]
    },
    {
       "hash":"0x05a7f5c5fce57346f59355184daa58822f97a32e4327fe6ef4a1c37dfd36f2f0",
       "parentHash":"0x64e89492b3ea72b9f9f0f4566e5198e19d7bfa583619c54c33872c7112aec9cd",
       "number":1,
       "timestamp":1603404102,
       "nonce":"0x0000000000000042",
       "difficulty":131072,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x688ddd1acc3cbccd9112018eac6c78744f43140d408128b8ed0392c7ee28966e",
             "blockHash":"0x05a7f5c5fce57346f59355184daa58822f97a32e4327fe6ef4a1c37dfd36f2f0",
             "blockNumber":1,
             "transactionIndex":0,
             "confirmations":16,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0x05bc67"
             },
             "to":null,
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":0,
             "data":"0x608060405234801561001057600080fd5b50600080546001600160a01b03191633178082556040516001600160a01b039190911691907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0908290a361056a806100696000396000f3fe608060405234801561001057600080fd5b50600436106100575760003560e01c8063715018a61461005c5780638da5cb5b146100665780639b2ea4bd1461008a578063bf40fac11461013b578063f2fde38b146101e1575b600080fd5b610064610207565b005b61006e6102b0565b604080516001600160a01b039092168252519081900360200190f35b610064600480360360408110156100a057600080fd5b8101906020810181356401000000008111156100bb57600080fd5b8201836020820111156100cd57600080fd5b803590602001918460018302840111640100000000831117156100ef57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250929550505090356001600160a01b031691506102bf9050565b61006e6004803603602081101561015157600080fd5b81019060208101813564010000000081111561016c57600080fd5b82018360208201111561017e57600080fd5b803590602001918460018302840111640100000000831117156101a057600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250929550610362945050505050565b610064600480360360208110156101f757600080fd5b50356001600160a01b0316610391565b6000546001600160a01b03163314610266576040805162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015290519081900360640190fd5b600080546040516001600160a01b03909116907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0908390a3600080546001600160a01b0319169055565b6000546001600160a01b031681565b6000546001600160a01b0316331461031e576040805162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015290519081900360640190fd5b806001600061032c85610490565b815260200190815260200160002060006101000a8154816001600160a01b0302191690836001600160a01b031602179055505050565b60006001600061037184610490565b81526020810191909152604001600020546001600160a01b031692915050565b6000546001600160a01b031633146103f0576040805162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015290519081900360640190fd5b6001600160a01b0381166104355760405162461bcd60e51b815260040180806020018281038252602d815260200180610508602d913960400191505060405180910390fd5b600080546040516001600160a01b03808516939216917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e091a3600080546001600160a01b0319166001600160a01b0392909216919091179055565b6000816040516020018082805190602001908083835b602083106104c55780518252601f1990920191602091820191016104a6565b6001836020036101000a03801982511681845116808217855250505050505090500191505060405160208183030381529060405280519060200120905091905056fe4f776e61626c653a206e6577206f776e65722063616e6e6f7420626520746865207a65726f2061646472657373a26469706673582212204367ffc2e6671623708150e2d0cff4c12cf566722a26b4748555d789953e2d2264736f6c63430007000033",
             "r":"0x2babe370e2e422a38386a5a96cd3bf16772ddbbf8c9dab8aadf4416fff557756",
             "s":"0x213ab994b50ed4a38e2de390f851d88cb66dd238a27d89246616b34eb8e859df",
             "v":62710,
             "creates":"0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA",
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0xb5903854b196abb9b7d1e3c2f5f8f9519b4fedefc21d72f3d92c74f128afcd46",
       "parentHash":"0x05a7f5c5fce57346f59355184daa58822f97a32e4327fe6ef4a1c37dfd36f2f0",
       "number":2,
       "timestamp":1603404103,
       "nonce":"0x0000000000000042",
       "difficulty":131136,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0xeae52be001bfea0b3d8de2de5edff5c0c39d26f6e6ab0fc6623cd24913dbf150",
             "blockHash":"0xb5903854b196abb9b7d1e3c2f5f8f9519b4fedefc21d72f3d92c74f128afcd46",
             "blockNumber":2,
             "transactionIndex":0,
             "confirmations":15,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0xaeab"
             },
             "to":"0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":1,
             "data":"0x9b2ea4bd000000000000000000000000000000000000000000000000000000000000004000000000000000000000000063fc2ad3d021a4d7e64323529a55a9442c444da0000000000000000000000000000000000000000000000000000000000000000d4f564d5f53657175656e63657200000000000000000000000000000000000000",
             "r":"0x530be666add21a30fe9a0fadee8072d4d8ccb2b80a9fe07c7e0591c5a1e5f375",
             "s":"0x5954b17cd0f6299ee54f20853bcfdfc870f2fc271f0433fd592b8b0fc0329aa4",
             "v":62709,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x7504e7104a2d80e810758c1cc2736028ffe5632a2ec988da740c93b3cfa03945",
       "parentHash":"0xb5903854b196abb9b7d1e3c2f5f8f9519b4fedefc21d72f3d92c74f128afcd46",
       "number":3,
       "timestamp":1603404104,
       "nonce":"0x0000000000000042",
       "difficulty":131200,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x79f4dc191018a37a8b71b4a5aff02ce501b0341fc72708c8095845e61be15c02",
             "blockHash":"0x7504e7104a2d80e810758c1cc2736028ffe5632a2ec988da740c93b3cfa03945",
             "blockNumber":3,
             "transactionIndex":0,
             "confirmations":14,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0xafb0"
             },
             "to":"0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":2,
             "data":"0x9b2ea4bd0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000420000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000224f564d5f4465636f6d7072657373696f6e507265636f6d70696c6541646472657373000000000000000000000000000000000000000000000000000000000000",
             "r":"0x388b6b7e4fe10128e92507042971b864edd535d8f3cb70c61247daed1dc734c1",
             "s":"0x6ad11366b26fa59ce73bc73fa320560f46ffb6df3142adbd16ebce0bcc3ba9f5",
             "v":62710,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x6587aa5da64944673741bdd3299c0d2471c2fdbfcd94713d511ab7240d127bbb",
       "parentHash":"0x7504e7104a2d80e810758c1cc2736028ffe5632a2ec988da740c93b3cfa03945",
       "number":4,
       "timestamp":1603404105,
       "nonce":"0x0000000000000042",
       "difficulty":131264,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x0584fbd34096ffc0fc96dbdf1d8ee574c3eda9e07bf85aed9862542416b1a007",
             "blockHash":"0x6587aa5da64944673741bdd3299c0d2471c2fdbfcd94713d511ab7240d127bbb",
             "blockNumber":4,
             "transactionIndex":0,
             "confirmations":13,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0x02e294"
             },
             "to":null,
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":3,
             "data":"0x608060405234801561001057600080fd5b50600080546001600160a01b03191633179055610213806100326000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063776d1a0114610077575b60015460408051602036601f8101829004820283018201909352828252610075936001600160a01b0316926000918190840183828082843760009201919091525061009d92505050565b005b6100756004803603602081101561008d57600080fd5b50356001600160a01b031661015d565b60006060836001600160a01b0316836040518082805190602001908083835b602083106100db5780518252601f1990920191602091820191016100bc565b6001836020036101000a0380198251168184511680821785525050505050509050019150506000604051808303816000865af19150503d806000811461013d576040519150601f19603f3d011682016040523d82523d6000602084013e610142565b606091505b5091509150811561015557805160208201f35b805160208201fd5b6000546001600160a01b031633141561019057600180546001600160a01b0319166001600160a01b0383161790556101da565b60015460408051602036601f81018290048202830182019093528282526101da936001600160a01b0316926000918190840183828082843760009201919091525061009d92505050565b5056fea2646970667358221220293887d48c4c1c34de868edf3e9a6be82327946c76d71f7c2023e67f556c6ecb64736f6c63430007000033",
             "r":"0x2e420851664bb81c0d5d0bd1a805661fc1f83922b92e1d9e0e57c9184eddec0e",
             "s":"0x5e00184c9b50ed54e714231af904caed92cf47ee309fc3604793d7d32a9f4988",
             "v":62709,
             "creates":"0x94BA4d5Ebb0e05A50e977FFbF6e1a1Ee3D89299c",
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x7a215669ab018e508473ef76b42d19cc4228cc5cc855089dc04cb22555fcf555",
       "parentHash":"0x6587aa5da64944673741bdd3299c0d2471c2fdbfcd94713d511ab7240d127bbb",
       "number":5,
       "timestamp":1603404106,
       "nonce":"0x0000000000000042",
       "difficulty":131328,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0xc8243d8b9bf79624f8930527a983be3581cb7d969ae76d3e5962f9db2be7b71a",
             "blockHash":"0x7a215669ab018e508473ef76b42d19cc4228cc5cc855089dc04cb22555fcf555",
             "blockNumber":5,
             "transactionIndex":0,
             "confirmations":12,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0xa93c"
             },
             "to":"0x94BA4d5Ebb0e05A50e977FFbF6e1a1Ee3D89299c",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":4,
             "data":"0x776d1a01000000000000000000000000e9a9a643588daa154de182f88a5b04e8745909c2",
             "r":"0x2de73fc5aec124cc9cf0fa54fc8492692a48b8921a32d31f46f5d431fdeea7a0",
             "s":"0x4e813ae50bc7705fb023d79429ed6cc92367cb5209460e6ea3904c014b06cd4d",
             "v":62709,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0xb4c01e6e867856dce9b451ba67273f86bdc584ff398ed5b8c46f70ea37b1002f",
       "parentHash":"0x7a215669ab018e508473ef76b42d19cc4228cc5cc855089dc04cb22555fcf555",
       "number":6,
       "timestamp":1603404107,
       "nonce":"0x0000000000000042",
       "difficulty":131392,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0xa5bf13c2638f1d99a26c4c7d867f1bc64a34ebf141524b35840c879536e69d69",
             "blockHash":"0xb4c01e6e867856dce9b451ba67273f86bdc584ff398ed5b8c46f70ea37b1002f",
             "blockNumber":6,
             "transactionIndex":0,
             "confirmations":11,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0xaeff"
             },
             "to":"0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":5,
             "data":"0x9b2ea4bd000000000000000000000000000000000000000000000000000000000000004000000000000000000000000094ba4d5ebb0e05a50e977ffbf6e1a1ee3d89299c00000000000000000000000000000000000000000000000000000000000000144f564d5f457865637574696f6e4d616e61676572000000000000000000000000",
             "r":"0x42a7ca8603050e58d948df4573374746fabc8542c3fabc1d3b03391c2e50ae3b",
             "s":"0x99b52df93bddbd15393ec9b6649c569ec9f65b1e7dda62b7a81baf32bd951a36",
             "v":62709,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x9f8a3aae03d0f8f769d5af64583c1ee32458bd5b63bf6436147e624ef3297c69",
       "parentHash":"0xb4c01e6e867856dce9b451ba67273f86bdc584ff398ed5b8c46f70ea37b1002f",
       "number":7,
       "timestamp":1603404108,
       "nonce":"0x0000000000000042",
       "difficulty":131456,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x967c90bfcb19090f0b73b0d0ca4d5f866160d7628d84412fada18abc50e3da69",
             "blockHash":"0x9f8a3aae03d0f8f769d5af64583c1ee32458bd5b63bf6436147e624ef3297c69",
             "blockNumber":7,
             "transactionIndex":0,
             "confirmations":10,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0x02e294"
             },
             "to":null,
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":6,
             "data":"0x608060405234801561001057600080fd5b50600080546001600160a01b03191633179055610213806100326000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063776d1a0114610077575b60015460408051602036601f8101829004820283018201909352828252610075936001600160a01b0316926000918190840183828082843760009201919091525061009d92505050565b005b6100756004803603602081101561008d57600080fd5b50356001600160a01b031661015d565b60006060836001600160a01b0316836040518082805190602001908083835b602083106100db5780518252601f1990920191602091820191016100bc565b6001836020036101000a0380198251168184511680821785525050505050509050019150506000604051808303816000865af19150503d806000811461013d576040519150601f19603f3d011682016040523d82523d6000602084013e610142565b606091505b5091509150811561015557805160208201f35b805160208201fd5b6000546001600160a01b031633141561019057600180546001600160a01b0319166001600160a01b0383161790556101da565b60015460408051602036601f81018290048202830182019093528282526101da936001600160a01b0316926000918190840183828082843760009201919091525061009d92505050565b5056fea2646970667358221220293887d48c4c1c34de868edf3e9a6be82327946c76d71f7c2023e67f556c6ecb64736f6c63430007000033",
             "r":"0x16f49916bda30884d49df7f83c60ca49899fd21311e4cd4b464ac52bfa722b40",
             "s":"0x3a2873207cbcba218ff71bb7e3916ea809c393c2407c03a70bbf4e393cbebfcb",
             "v":62709,
             "creates":"0x956dA338C1518a7FB213042b70c60c021aeBd554",
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x808315e40a80d00bf171b8ba924b451fb7936d446177c9e3545303b2ef830801",
       "parentHash":"0x9f8a3aae03d0f8f769d5af64583c1ee32458bd5b63bf6436147e624ef3297c69",
       "number":8,
       "timestamp":1603404109,
       "nonce":"0x0000000000000042",
       "difficulty":131520,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x3dabdb84873e50c3b26df28ead1c053451a340b20d945a8cf8bddf5b5ff11775",
             "blockHash":"0x808315e40a80d00bf171b8ba924b451fb7936d446177c9e3545303b2ef830801",
             "blockNumber":8,
             "transactionIndex":0,
             "confirmations":9,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0xa93c"
             },
             "to":"0x956dA338C1518a7FB213042b70c60c021aeBd554",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":7,
             "data":"0x776d1a01000000000000000000000000048b45c16e9631d3f630106c6086ec21a30cdf60",
             "r":"0x553525e3656dfb41299a9f2a0d1a0058445e017bb98992de6d9b762203cf975a",
             "s":"0x74b9f344513fa2881cfcec5a4c20027bb905de5c52a8e584da6767d946cd465d",
             "v":62710,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0xed13ac7f6bf594309a325c1558e4a3032f745e66ee0ffc2f95a701562b53dd13",
       "parentHash":"0x808315e40a80d00bf171b8ba924b451fb7936d446177c9e3545303b2ef830801",
       "number":9,
       "timestamp":1603404110,
       "nonce":"0x0000000000000042",
       "difficulty":131584,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x2b9e216b390804c731f3ffc0630e522865bff35d0c539f377d204e94a346eddd",
             "blockHash":"0xed13ac7f6bf594309a325c1558e4a3032f745e66ee0ffc2f95a701562b53dd13",
             "blockNumber":9,
             "transactionIndex":0,
             "confirmations":8,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0xaf2f"
             },
             "to":"0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":8,
             "data":"0x9b2ea4bd0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000956da338c1518a7fb213042b70c60c021aebd55400000000000000000000000000000000000000000000000000000000000000184f564d5f5374617465436f6d6d69746d656e74436861696e0000000000000000",
             "r":"0x6318ce7714d7aefc9add30ebcf9d657a01b50c308c689c6f7d5567d806b6914e",
             "s":"0xfe4d90d895f4b18b89309d768a383334bf5d622d367051079ac0e401b23960e0",
             "v":62709,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x65f3831f69d1e746f30d07a3def1eefdcc6650ee0714cbac795c2be3ce9430a2",
       "parentHash":"0xed13ac7f6bf594309a325c1558e4a3032f745e66ee0ffc2f95a701562b53dd13",
       "number":10,
       "timestamp":1603404111,
       "nonce":"0x0000000000000042",
       "difficulty":131648,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0xa77aca829b5bc91a16c11efb4dee280345851fccbd83b1469f746349894a46ab",
             "blockHash":"0x65f3831f69d1e746f30d07a3def1eefdcc6650ee0714cbac795c2be3ce9430a2",
             "blockNumber":10,
             "transactionIndex":0,
             "confirmations":7,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0x24f27e"
             },
             "to":null,
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":9,
             "data":"0x12345678",
             "r":"0x06e79a060942823dc5b328a5b059b58cf42372c03617122139deb5b7844c043d",
             "s":"0xfabd07fab3f36816397917ae8c048a4675d34d4ca3f7b06ca6595796a159d359",
             "v":62710,
             "creates":"0x6454C9d69a4721feBA60e26A367bD4D56196Ee7c",
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x709bb2a24ac6120a12bf56e806cfcf3ceab71a41408b0685892800eebc4e3a45",
       "parentHash":"0x65f3831f69d1e746f30d07a3def1eefdcc6650ee0714cbac795c2be3ce9430a2",
       "number":11,
       "timestamp":1603404112,
       "nonce":"0x0000000000000042",
       "difficulty":131712,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x2b5cd67036954f3c4cce09951c076adb3d0517750167e712562abba7910bd536",
             "blockHash":"0x709bb2a24ac6120a12bf56e806cfcf3ceab71a41408b0685892800eebc4e3a45",
             "blockNumber":11,
             "transactionIndex":0,
             "confirmations":6,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0x0f4240"
             },
             "to":"0x6454C9d69a4721feBA60e26A367bD4D56196Ee7c",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":10,
             "data":"0x6fee07e00000000000000000000000000101010101010101010101010101010101010101000000000000000000000000000000000000000000000000000000000000c350000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000201111111111111111111111111111111111111111111111111111111111111111",
             "r":"0x7538f2153c482a762f133f1438b30c8874887f6da52f04adeba5cc65632ee661",
             "s":"0x3dc3a219d164b159ad61f874ccb5c75766cb1d73954196eaba142d890b299d0e",
             "v":62710,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x13e683738bac942ae739e45b0a0e451b20d7e986d9463a2dac0acf2015e8d09b",
       "parentHash":"0x709bb2a24ac6120a12bf56e806cfcf3ceab71a41408b0685892800eebc4e3a45",
       "number":12,
       "timestamp":1603404113,
       "nonce":"0x0000000000000042",
       "difficulty":131776,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x339282191d55c5b9fc53d68f199c53ea9b946c9975ed34f15312acd8a70054f7",
             "blockHash":"0x13e683738bac942ae739e45b0a0e451b20d7e986d9463a2dac0acf2015e8d09b",
             "blockNumber":12,
             "transactionIndex":0,
             "confirmations":5,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0x0f4240"
             },
             "to":"0x6454C9d69a4721feBA60e26A367bD4D56196Ee7c",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":11,
             "data":"0x6fee07e00000000000000000000000000101010101010101010101010101010101010101000000000000000000000000000000000000000000000000000000000000c350000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000202222222222222222222222222222222222222222222222222222222222222222",
             "r":"0x4ab1b83146fbfa4f0cce0110149c2c52e57971ce7cbe5b97a3fd3086bf9f0935",
             "s":"0x11dac6b6c1e1d66a89d833ddd230120e6ecdedf49ae9fb38496f4385cb80057d",
             "v":62709,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x382c468636dafe28d7b3aad0f0521c2e7b9395fc0146b16502d85fa2a49ffe7b",
       "parentHash":"0x13e683738bac942ae739e45b0a0e451b20d7e986d9463a2dac0acf2015e8d09b",
       "number":13,
       "timestamp":1603404114,
       "nonce":"0x0000000000000042",
       "difficulty":131840,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x8c260f877daaaaeba12b41da6763ce2a641a55ec2e545c4dcbc211b340480e93",
             "blockHash":"0x382c468636dafe28d7b3aad0f0521c2e7b9395fc0146b16502d85fa2a49ffe7b",
             "blockNumber":13,
             "transactionIndex":0,
             "confirmations":4,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0x0f4240"
             },
             "to":"0x6454C9d69a4721feBA60e26A367bD4D56196Ee7c",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":12,
             "data":"0x6fee07e00000000000000000000000000101010101010101010101010101010101010101000000000000000000000000000000000000000000000000000000000000c350000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000203333333333333333333333333333333333333333333333333333333333333333",
             "r":"0x52f5316fc04aafc95110ac6be222ed656cd0a4ace50fac3e09384408c8b7e32a",
             "s":"0x03de7779650526057629e842c45017cfd2dc19137d309cd99f244c0ce9b13186",
             "v":62709,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x5e5f592a3c96a45bc7fceea8681d066af84709dc155d305fe4e3d68f0bb7bd63",
       "parentHash":"0x382c468636dafe28d7b3aad0f0521c2e7b9395fc0146b16502d85fa2a49ffe7b",
       "number":14,
       "timestamp":1603404115,
       "nonce":"0x0000000000000042",
       "difficulty":131904,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x2fbd6a3a19c6e5778762619ecfd849b4c3beb35bce0c99b1f357077f3380fa9e",
             "blockHash":"0x5e5f592a3c96a45bc7fceea8681d066af84709dc155d305fe4e3d68f0bb7bd63",
             "blockNumber":14,
             "transactionIndex":0,
             "confirmations":3,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0x0f4240"
             },
             "to":"0x6454C9d69a4721feBA60e26A367bD4D56196Ee7c",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":13,
             "data":"0x6fee07e00000000000000000000000000101010101010101010101010101010101010101000000000000000000000000000000000000000000000000000000000000c350000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000204444444444444444444444444444444444444444444444444444444444444444",
             "r":"0x546c5a59b1753dd9b9f1f7ff0ae10f6f7fe07bfadf9882e140a021b0ca1a8ab4",
             "s":"0x8b6c62b492ea8cf8e7dc8e8236ce7283a24e18370ce7950bda5f450fb7993547",
             "v":62709,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    },
    {
       "hash":"0x13be1ecbdbaae00332acaa341ea3168781b112e7aff368a8bab060fa102085f4",
       "parentHash":"0x5e5f592a3c96a45bc7fceea8681d066af84709dc155d305fe4e3d68f0bb7bd63",
       "number":15,
       "timestamp":1603404116,
       "nonce":"0x0000000000000042",
       "difficulty":131968,
       "gasLimit":{
          "type":"BigNumber",
          "hex":"0x02625a00"
       },
       "gasUsed":{
          "type":"BigNumber",
          "hex":"0x00"
       },
       "miner":"0xC014BA5EC014ba5ec014Ba5EC014ba5Ec014bA5E",
       "extraData":"0x",
       "transactions":[
          {
             "hash":"0x3a60f459be600341f831b6e9b6b75a242cd31d1e4ae6b0bbd763a6b56054ef7b",
             "blockHash":"0x13be1ecbdbaae00332acaa341ea3168781b112e7aff368a8bab060fa102085f4",
             "blockNumber":15,
             "transactionIndex":0,
             "confirmations":2,
             "from":"0x17ec8597ff92C3F44523bDc65BF0f1bE632917ff",
             "gasPrice":{
                "type":"BigNumber",
                "hex":"0x01dcd65000"
             },
             "gasLimit":{
                "type":"BigNumber",
                "hex":"0x0f4240"
             },
             "to":"0x6454C9d69a4721feBA60e26A367bD4D56196Ee7c",
             "value":{
                "type":"BigNumber",
                "hex":"0x00"
             },
             "nonce":14,
             "data":"0x6fee07e00000000000000000000000000101010101010101010101010101010101010101000000000000000000000000000000000000000000000000000000000000c350000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000205555555555555555555555555555555555555555555555555555555555555555",
             "r":"0x3b214ae35181aaf5542f970325ef36ca881db2fc1b145524534a4fb6885b05d7",
             "s":"0xdf4fd847db5d5a246f4cedd2dee14e15e70ff469a7d88cdc86efa7c6d61d7cad",
             "v":62709,
             "creates":null,
             "l1BlockNumber":"1",
             "l1TxOrigin":"0x3333333333333333333333333333333333333333",
             "rawTransaction":"0x420420",
             "signatureHashType":"0",
             "queueOrigin":"sequencer",
             "chainId":31337
          }
       ]
    }
 ]
`)

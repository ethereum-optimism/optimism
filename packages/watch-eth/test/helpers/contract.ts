import { web3 } from './ethereum'

const ABI = [
  {
    constant: false,
    inputs: [],
    name: 'test',
    outputs: [],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    anonymous: false,
    inputs: [{ indexed: false, name: '_value', type: 'uint256' }],
    name: 'TestEvent',
    type: 'event',
  },
]
const BYTECODE =
  '0x6080604052348015600f57600080fd5b5060b88061001e6000396000f3fe6080604052600436106039576000357c010000000000000000000000000000000000000000000000000000000090048063f8a8fd6d14603e575b600080fd5b348015604957600080fd5b5060506052565b005b7f1440c4dd67b4344ea1905ec0318995133b550f168b4ee959a0da6b503d7d2414607b6040518082815260200191505060405180910390a156fea165627a7a7230582064f91b684a76913a3071227bf464191e93dbf60cd80e7dd64ab1ddedcbab54c50029'

export class DummyContract {
  private contract = new web3.eth.Contract(ABI)

  get address(): string {
    return this.contract.options.address
  }

  get abi(): any[] {
    return ABI
  }

  public async deploy(): Promise<string> {
    const accounts = await web3.eth.getAccounts()
    await this.contract
      .deploy({
        data: BYTECODE,
        arguments: [],
      })
      .send({
        from: accounts[0],
        gas: 1000000,
        gasPrice: '1',
      })
    return this.contract.options.address
  }

  public async createEvents(total: number): Promise<void> {
    const accounts = await web3.eth.getAccounts()
    for (let i = 0; i < total; i++) {
      await this.contract.methods.test().send({
        from: accounts[0],
      })
    }
  }
}

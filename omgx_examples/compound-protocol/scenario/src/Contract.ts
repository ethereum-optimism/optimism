import * as path from 'path';
import * as crypto from 'crypto';
import { World } from './World';
import { Invokation } from './Invokation';
import { readFile } from './File';
import { AbiItem } from 'web3-utils';

export interface Raw {
  data: string
  topics: string[]
}

export interface Event {
  event: string
  signature: string | null
  address: string
  returnValues: object
  logIndex: number
  transactionIndex: number
  blockHash: string
  blockNumber: number
  raw: Raw
}

export interface Contract {
  address: string
  _address: string
  name: string
  methods: any
  _jsonInterface: AbiItem[]
  constructorAbi?: string
  getPastEvents: (event: string, options: { filter: object, fromBlock: number, toBlock: number | string }) => Event[]
}

function randomAddress(): string {
  return crypto.randomBytes(20).toString('hex');
}

class ContractStub {
  name: string;
  test: boolean

  constructor(name: string, test: boolean) {
    this.name = name;
    this.test = test;
  }

  async deploy<T>(world: World, from: string, args: any[]): Promise<Invokation<T>> {
    // XXXS Consider opts
    // ( world.web3.currentProvider && typeof(world.web3.currentProvider) !== 'string' && world.web3.currentProvider.opts ) || 
    const opts = { from: from };

    let invokationOpts = world.getInvokationOpts(opts);

    const networkContractABI = await world.saddle.abi(this.name);
    const constructorAbi = networkContractABI.find((x) => x.type === 'constructor');
    let inputs;

    if (constructorAbi) {
      inputs = constructorAbi.inputs;
    } else {
      inputs = [];
    }

    try {
      let contract;
      let receipt;

      if (world.dryRun) {
        let addr = randomAddress();
        console.log(`Dry run: Deploying ${this.name} at fake address ${addr}`);
        contract = new world.web3.eth.Contract(<any>networkContractABI, addr)
        receipt = {
          blockNumber: -1,
          transactionHash: "0x",
          events: {}
        };
      } else {
        ({ contract, receipt } = await world.saddle.deployFull(this.name, args, invokationOpts, world.web3));
        contract.constructorAbi = world.web3.eth.abi.encodeParameters(inputs, args);;
      }

      return new Invokation<T>(contract, receipt, null, null);
    } catch (err) {
      return new Invokation<T>(null, null, err, null);
    }
  }

  async at<T>(world: World, address: string): Promise<T> {
    const networkContractABI = await world.saddle.abi(this.name);

    // XXXS unknown?
    return <T><unknown>(new world.web3.eth.Contract(<any>networkContractABI, address));
  }
}

export function getContract(name: string): ContractStub {
  return new ContractStub(name, false);
}

export function getTestContract(name: string): ContractStub {
  return new ContractStub(name, true);
}

export function setContractName(name: string, contract: Contract): Contract {
  contract.name = name;

  return contract;
}

export async function getPastEvents(world: World, contract: Contract, name: string, event: string, filter: object = {}): Promise<Event[]> {
  const block = world.getIn(['contractData', 'Blocks', name]);
  if (!block) {
    throw new Error(`Cannot get events when missing deploy block for ${name}`);
  }

  return await contract.getPastEvents(event, { filter: filter, fromBlock: block, toBlock: 'latest' });
}

export async function decodeCall(world: World, contract: Contract, input: string): Promise<World> {
  if (input.slice(0, 2) === '0x') {
    input = input.slice(2);
  }

  let functionSignature = input.slice(0, 8);
  let argsEncoded = input.slice(8);

  let funsMapped = contract._jsonInterface.reduce((acc, fun) => {
    if (fun.type === 'function') {
      let functionAbi = `${fun.name}(${(fun.inputs || []).map((i) => i.type).join(',')})`;
      let sig = world.web3.utils.sha3(functionAbi).slice(2, 10);

      return {
        ...acc,
        [sig]: fun
      };
    } else {
      return acc;
    }
  }, {});

  let abi = funsMapped[functionSignature];

  if (!abi) {
    throw new Error(`Cannot find function matching signature ${functionSignature}`);
  }

  let decoded = world.web3.eth.abi.decodeParameters(abi.inputs, argsEncoded);

  const args = abi.inputs.map((input) => {
    return `${input.name}=${decoded[input.name]}`;
  });
  world.printer.printLine(`\n${contract.name}.${abi.name}(\n\t${args.join("\n\t")}\n)`);

  return world;
}

// XXXS Handle
async function getNetworkContract(world: World, name: string): Promise<{ abi: any[], bin: string }> {
  let basePath = world.basePath || ""
  let network = world.network || ""

  let pizath = (name, ext) => path.join(basePath, '.build', `contracts.json`);
  let abi, bin;
  if (network == 'coverage') {
    let json = await readFile(world, pizath(name, 'json'), null, JSON.parse);
    abi = json.abi;
    bin = json.bytecode.substr(2);
  } else {
    let { networkContracts } = await getNetworkContracts(world);
    let networkContract = networkContracts[name];
    abi = JSON.parse(networkContract.abi);
    bin = networkContract.bin;
  }
  if (!bin) {
    throw new Error(`no bin for contract ${name} ${network}`)
  }
  return {
    abi: abi,
    bin: bin
  }
}

export async function getNetworkContracts(world: World): Promise<{ networkContracts: object, version: string }> {
  let basePath = world.basePath || ""
  let network = world.network || ""

  let contractsPath = path.join(basePath, '.build', `contracts.json`)
  let fullContracts = await readFile(world, contractsPath, null, JSON.parse);
  let version = fullContracts.version;
  let networkContracts = Object.entries(fullContracts.contracts).reduce((acc, [k, v]) => {
    let [path, contractName] = k.split(':');

    return {
      ...acc,
      [contractName]: {
        ...<object>v, /// XXXS TODO
        path: path
      }
    };
  }, {});

  return {
    networkContracts,
    version
  };
}

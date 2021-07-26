import { fromJS, Map } from 'immutable';
import { World } from './World';
import { Invokation } from './Invokation';
import { Contract, setContractName } from './Contract';
import { getNetworkPath, readFile, writeFile } from './File';
import { AbiItem } from 'web3-utils';

type Networks = Map<string, any>;

interface ExtraData {
  index: string[];
  data: object | string | number;
}

export function parseNetworkFile(data: string | object): Networks {
  return fromJS(typeof data === 'string' ? JSON.parse(data) : data);
}

function serializeNetworkFile(networks: Networks): string {
  return JSON.stringify(networks.toJSON(), null, 4);
}

function readNetworkFile(world: World, isABI: boolean): Promise<Networks> {
  return readFile(
    world,
    getNetworkPath(world.basePath, world.network, isABI ? '-abi' : ''),
    Map({}),
    parseNetworkFile
  );
}

function writeNetworkFile(world: World, networks: Networks, isABI: boolean): Promise<World> {
  return writeFile(
    world,
    getNetworkPath(world.basePath, world.network, isABI ? '-abi' : ''),
    serializeNetworkFile(networks)
  );
}

export function storeContract(world: World, contract: Contract, name: string, extraData: ExtraData[]): World {
  contract = setContractName(name, contract);

  world = world.set('lastContract', contract);
  world = world.setIn(['contractIndex', contract._address.toLowerCase()], contract);
  world = updateEventDecoder(world, contract);

  world = world.update('contractData', contractData => {
    return extraData.reduce((acc, { index, data }) => {
      if (typeof data !== 'string' && typeof data !== 'number') {
        // Store extra data as an immutable
        data = Map(<any>data);
      }

      return acc.setIn(index, data);
    }, contractData);
  });

  return world;
}

export async function saveContract<T>(
  world: World,
  contract: Contract,
  name: string,
  extraData: ExtraData[]
): Promise<World> {
  let networks = await readNetworkFile(world, false);
  let networksABI = await readNetworkFile(world, true);

  networks = extraData.reduce((acc, { index, data }) => acc.setIn(index, data), networks);
  networksABI = networksABI.set(name, contract._jsonInterface);

  // Don't write during a dry-run
  if (!world.dryRun) {
    world = await writeNetworkFile(world, networks, false);
    world = await writeNetworkFile(world, networksABI, true);
  }

  return world;
}

// Merges a contract into another, which is important for delegation
export async function mergeContractABI(
  world: World,
  targetName: string,
  contractTarget: Contract,
  a: string,
  b: string
): Promise<World> {
  let networks = await readNetworkFile(world, false);
  let networksABI = await readNetworkFile(world, true);
  let aABI = networksABI.get(a);
  let bABI = networksABI.get(b);

  if (!aABI) {
    throw new Error(`Missing contract ABI for ${a}`);
  }

  if (!bABI) {
    throw new Error(`Missing contract ABI for ${b}`);
  }

  const itemBySig: { [key: string]: AbiItem } = {};
  for (let item of aABI.toJS().concat(bABI.toJS())) {
    itemBySig[item.signature] = item;
  }
  const fullABI = Object.values(itemBySig);

  // Store Comptroller address
  networks = networks.setIn(['Contracts', targetName], contractTarget._address);
  world = world.setIn(['contractData', 'Contracts', targetName], contractTarget._address);

  networksABI = networksABI.set(targetName, fullABI);

  let mergedContract = new world.web3.eth.Contract(fullABI, contractTarget._address, {});

  /// XXXS
  world = world.setIn(
    ['contractIndex', contractTarget._address.toLowerCase()],
    setContractName(targetName, <Contract><unknown>mergedContract)
  );

  // Don't write during a dry-run
  if (!world.dryRun) {
    world = await writeNetworkFile(world, networks, false);
    world = await writeNetworkFile(world, networksABI, true);
  }

  return world;
}

export async function loadContracts(world: World): Promise<[World, string[]]> {
  let networks = await readNetworkFile(world, false);
  let networksABI = await readNetworkFile(world, true);

  return loadContractData(world, networks, networksABI);
}

function updateEventDecoder(world: World, contract: any) {
  const updatedEventDecoder = contract._jsonInterface
    .filter(i => i.type == 'event')
    .reduce((accum, event) => {
      const { anonymous, inputs, signature } = event;
      return {
        ...accum,
        [signature]: log => {
          let argTopics = anonymous ? log.topics : log.topics.slice(1);
          return world.web3.eth.abi.decodeLog(inputs, log.data, argTopics);
        }
      };
    }, world.eventDecoder);

  return world.set('eventDecoder', updatedEventDecoder)
}

export async function loadContractData(
  world: World,
  networks: Networks,
  networksABI: Networks
): Promise<[World, string[]]> {
  // Pull off contracts value and the rest is "extra"
  let contractInfo: string[] = [];
  let contracts = networks.get('Contracts') || Map({});

  world = contracts.reduce((world: World, address: string, name: string) => {
    let abi: AbiItem[] = networksABI.has(name) ? networksABI.get(name).toJS() : [];
    let contract = new world.web3.eth.Contract(abi, address, {});

    world = updateEventDecoder(world, contract);

    contractInfo.push(`${name}: ${address}`);

    // Store the contract
    // XXXS
    return world.setIn(['contractIndex', (<any>contract)._address.toLowerCase()], setContractName(name, <Contract><unknown>contract));
  }, world);

  world = world.update('contractData', contractData => contractData.mergeDeep(networks));

  return [world, contractInfo];
}

export async function storeAndSaveContract<T>(
  world: World,
  contract: Contract,
  name: string,
  invokation: Invokation<T> | null,
  extraData: ExtraData[]
): Promise<World> {
  extraData.push({ index: ['Contracts', name], data: contract._address });
  if (contract.constructorAbi) {
    extraData.push({ index: ['Constructors', name], data: contract.constructorAbi });
  }
  if (invokation && invokation.receipt) {
    extraData.push({ index: ['Blocks', name], data: invokation.receipt.blockNumber });
  }

  world = storeContract(world, contract, name, extraData);
  world = await saveContract(world, contract, name, extraData);

  return world;
}

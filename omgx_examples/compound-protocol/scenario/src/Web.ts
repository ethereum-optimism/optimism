import { parse } from './Parser';
import { World, initWorld } from './World';
import { throwExpect } from './Assert';
import { CallbackPrinter } from './Printer';
import { runCommand } from './Runner';
import { loadContractData, parseNetworkFile } from './Networks';
import Web3 from 'web3';
import { Saddle } from 'eth-saddle';

function networkFromId(id: number) {
  switch (id) {
    case 0:
      return 'olympic';

    case 1:
      return 'mainnet';

    case 2:
      return 'morden';

    case 3:
      return 'ropsten';

    case 4:
      return 'rinkeby';

    case 5:
      return 'goerli';

    case 8:
      return 'ubiq';

    case 42:
      return 'kovan';

    case 77:
      return 'sokol';

    case 99:
      return 'core';

    case 999:
      return 'development';

    default:
      return '';
  }
}

export async function webWorld(
  web3: Web3,
  networksData: string,
  networksABIData: string,
  printerCallback: (message: any) => void
): Promise<World> {
  let printer = new CallbackPrinter(printerCallback);
  let accounts;
  if (web3.currentProvider && typeof(web3.currentProvider) !== 'string') {
    // XXXS
    accounts = [(<any>web3.currentProvider).address];
  }

  const networkId = await (web3 as any).net.getId();
  const network: string = networkFromId(networkId);

  // XXXS
  const saddle = <Saddle><unknown>{
    web3: web3
  };

  let world = await initWorld(throwExpect, printer, web3, saddle, network, accounts, null, null);

  let networks = parseNetworkFile(networksData);
  let networksABI = parseNetworkFile(networksABIData);

  [world] = await loadContractData(world, networks, networksABI);
  // world = loadInvokationOpts(world);
  // world = loadVerbose(world);
  // world = loadDryRun(world);
  // world = await loadSettings(world);

  return world;
}

export async function webParse(world: World, line: string): Promise<World> {
  return runCommand(world, line, {});
}

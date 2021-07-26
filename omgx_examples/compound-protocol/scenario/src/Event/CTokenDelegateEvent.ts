import { Event } from '../Event';
import { addAction, describeUser, World } from '../World';
import { decodeCall, getPastEvents } from '../Contract';
import { CToken, CTokenScenario } from '../Contract/CToken';
import { CErc20Delegate } from '../Contract/CErc20Delegate'
import { invoke, Sendable } from '../Invokation';
import {
  getAddressV,
  getEventV,
  getExpNumberV,
  getNumberV,
  getStringV,
  getBoolV
} from '../CoreValue';
import {
  AddressV,
  BoolV,
  EventV,
  NothingV,
  NumberV,
  StringV
} from '../Value';
import { Arg, Command, View, processCommandEvent } from '../Command';
import { getCTokenDelegateData } from '../ContractLookup';
import { buildCTokenDelegate } from '../Builder/CTokenDelegateBuilder';
import { verify } from '../Verify';

async function genCTokenDelegate(world: World, from: string, event: Event): Promise<World> {
  let { world: nextWorld, cTokenDelegate, delegateData } = await buildCTokenDelegate(world, from, event);
  world = nextWorld;

  world = addAction(
    world,
    `Added cToken ${delegateData.name} (${delegateData.contract}) at address ${cTokenDelegate._address}`,
    delegateData.invokation
  );

  return world;
}

async function verifyCTokenDelegate(world: World, cTokenDelegate: CErc20Delegate, name: string, contract: string, apiKey: string): Promise<World> {
  if (world.isLocalNetwork()) {
    world.printer.printLine(`Politely declining to verify on local network: ${world.network}.`);
  } else {
    await verify(world, apiKey, name, contract, cTokenDelegate._address);
  }

  return world;
}

export function cTokenDelegateCommands() {
  return [
    new Command<{ cTokenDelegateParams: EventV }>(`
        #### Deploy

        * "CTokenDelegate Deploy ...cTokenDelegateParams" - Generates a new CTokenDelegate
          * E.g. "CTokenDelegate Deploy CDaiDelegate cDAIDelegate"
      `,
      "Deploy",
      [new Arg("cTokenDelegateParams", getEventV, { variadic: true })],
      (world, from, { cTokenDelegateParams }) => genCTokenDelegate(world, from, cTokenDelegateParams.val)
    ),
    new View<{ cTokenDelegateArg: StringV, apiKey: StringV }>(`
        #### Verify

        * "CTokenDelegate <cTokenDelegate> Verify apiKey:<String>" - Verifies CTokenDelegate in Etherscan
          * E.g. "CTokenDelegate cDaiDelegate Verify "myApiKey"
      `,
      "Verify",
      [
        new Arg("cTokenDelegateArg", getStringV),
        new Arg("apiKey", getStringV)
      ],
      async (world, { cTokenDelegateArg, apiKey }) => {
        let [cToken, name, data] = await getCTokenDelegateData(world, cTokenDelegateArg.val);

        return await verifyCTokenDelegate(world, cToken, name, data.get('contract')!, apiKey.val);
      },
      { namePos: 1 }
    ),
  ];
}

export async function processCTokenDelegateEvent(world: World, event: Event, from: string | null): Promise<World> {
  return await processCommandEvent<any>("CTokenDelegate", cTokenDelegateCommands(), world, event, from);
}

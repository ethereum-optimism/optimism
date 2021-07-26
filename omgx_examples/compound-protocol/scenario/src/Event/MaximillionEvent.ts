import {Event} from '../Event';
import {addAction, describeUser, World} from '../World';
import {Maximillion} from '../Contract/Maximillion';
import {invoke} from '../Invokation';
import {
  getAddressV,
  getEventV,
  getStringV
} from '../CoreValue';
import {
  AddressV,
  EventV,
  NumberV,
  StringV
} from '../Value';
import {Arg, Command, View, processCommandEvent} from '../Command';
import {buildMaximillion} from '../Builder/MaximillionBuilder';
import {getMaximillion} from '../ContractLookup';
import {verify} from '../Verify';

function showTrxValue(world: World): string {
  return new NumberV(world.trxInvokationOpts.get('value')).show();
}

async function genMaximillion(world: World, from: string, params: Event): Promise<World> {
  let {world: nextWorld, maximillion, maximillionData} = await buildMaximillion(world, from, params);
  world = nextWorld;

  world = addAction(
    world,
    `Added Maximillion (${maximillionData.description}) at address ${maximillion._address}`,
    maximillionData.invokation
  );

  return world;
}

async function verifyMaximillion(world: World, maximillion: Maximillion, apiKey: string): Promise<World> {
  if (world.isLocalNetwork()) {
    world.printer.printLine(`Politely declining to verify on local network: ${world.network}.`);
  } else {
    await verify(world, apiKey, "Maximillion", "Maximillion", maximillion._address);
  }

  return world;
}

async function repayBehalf(world: World, from: string, maximillion: Maximillion, behalf: string): Promise<World> {
  let showAmount = showTrxValue(world);
  let invokation = await invoke(world, maximillion.methods.repayBehalf(behalf), from);

  world = addAction(
    world,
    `Maximillion: ${describeUser(world, from)} repays ${showAmount} of borrow on behalf of ${describeUser(world, behalf)}`,
    invokation
  );

  return world;
}

export function maximillionCommands() {
  return [
    new Command<{maximillionParams: EventV}>(`
        #### Deploy

        * "Maximillion Deploy ...maximillionParams" - Generates a new Maximillion
          * E.g. "Maximillion Deploy"
      `,
      "Deploy",
      [new Arg("maximillionParams", getEventV, {variadic: true})],
      (world, from, {maximillionParams}) => genMaximillion(world, from, maximillionParams.val)
    ),
    new View<{maximillion: Maximillion, apiKey: StringV}>(`
        #### Verify

        * "Maximillion Verify apiKey:<String>" - Verifies Maximillion in Etherscan
          * E.g. "Maximillion Verify "myApiKey"
      `,
      "Verify",
      [
        new Arg("maximillion", getMaximillion, {implicit: true}),
        new Arg("apiKey", getStringV)
      ],
      (world, {maximillion, apiKey}) => verifyMaximillion(world, maximillion, apiKey.val)
    ),
    new Command<{maximillion: Maximillion, behalf: AddressV}>(`
        #### RepayBehalf

        * "RepayBehalf behalf:<User>" - Repays up to given value of given user's borrow
          * E.g. "(Trx Value 1.0e18 (Maximillion RepayBehalf Geoff))"
      `,
      "RepayBehalf",
      [
        new Arg("maximillion", getMaximillion, {implicit: true}),
        new Arg("behalf", getAddressV)
      ],
      (world, from, {maximillion, behalf}) => repayBehalf(world, from, maximillion, behalf.val)
    )
  ];
}

export async function processMaximillionEvent(world: World, event: Event, from: string | null): Promise<World> {
  return await processCommandEvent<any>("Maximillion", maximillionCommands(), world, event, from);
}

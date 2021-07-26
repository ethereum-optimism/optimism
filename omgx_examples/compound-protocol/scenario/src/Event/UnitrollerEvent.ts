import { Event } from '../Event';
import { addAction, describeUser, World } from '../World';
import { Unitroller } from '../Contract/Unitroller';
import { ComptrollerImpl } from '../Contract/ComptrollerImpl';
import { invoke } from '../Invokation';
import { getEventV, getStringV, getAddressV } from '../CoreValue';
import { EventV, StringV, AddressV } from '../Value';
import { Arg, Command, View, processCommandEvent } from '../Command';
import { ComptrollerErrorReporter } from '../ErrorReporter';
import { buildUnitroller } from '../Builder/UnitrollerBuilder';
import { getComptrollerImpl, getUnitroller } from '../ContractLookup';
import { verify } from '../Verify';

async function genUnitroller(world: World, from: string, params: Event): Promise<World> {
  let { world: nextWorld, unitroller, unitrollerData } = await buildUnitroller(world, from, params);
  world = nextWorld;

  world = addAction(
    world,
    `Added Unitroller (${unitrollerData.description}) at address ${unitroller._address}`,
    unitrollerData.invokation
  );

  return world;
}

async function verifyUnitroller(world: World, unitroller: Unitroller, apiKey: string): Promise<World> {
  if (world.isLocalNetwork()) {
    world.printer.printLine(`Politely declining to verify on local network: ${world.network}.`);
  } else {
    await verify(world, apiKey, 'Unitroller', 'Unitroller', unitroller._address);
  }

  return world;
}

async function acceptAdmin(world: World, from: string, unitroller: Unitroller): Promise<World> {
  let invokation = await invoke(world, unitroller.methods._acceptAdmin(), from, ComptrollerErrorReporter);

  world = addAction(world, `Accept admin as ${from}`, invokation);

  return world;
}

async function setPendingAdmin(
  world: World,
  from: string,
  unitroller: Unitroller,
  pendingAdmin: string
): Promise<World> {
  let invokation = await invoke(
    world,
    unitroller.methods._setPendingAdmin(pendingAdmin),
    from,
    ComptrollerErrorReporter
  );

  world = addAction(world, `Set pending admin to ${pendingAdmin}`, invokation);

  return world;
}

async function setPendingImpl(
  world: World,
  from: string,
  unitroller: Unitroller,
  comptrollerImpl: ComptrollerImpl
): Promise<World> {
  let invokation = await invoke(
    world,
    unitroller.methods._setPendingImplementation(comptrollerImpl._address),
    from,
    ComptrollerErrorReporter
  );

  world = addAction(world, `Set pending comptroller impl to ${comptrollerImpl.name}`, invokation);

  return world;
}

export function unitrollerCommands() {
  return [
    new Command<{ unitrollerParams: EventV }>(
      `
        #### Deploy

        * "Unitroller Deploy ...unitrollerParams" - Generates a new Unitroller
          * E.g. "Unitroller Deploy"
      `,
      'Deploy',
      [new Arg('unitrollerParams', getEventV, { variadic: true })],
      (world, from, { unitrollerParams }) => genUnitroller(world, from, unitrollerParams.val)
    ),
    new View<{ unitroller: Unitroller; apiKey: StringV }>(
      `
        #### Verify

        * "Unitroller Verify apiKey:<String>" - Verifies Unitroller in Etherscan
          * E.g. "Unitroller Verify "myApiKey"
      `,
      'Verify',
      [new Arg('unitroller', getUnitroller, { implicit: true }), new Arg('apiKey', getStringV)],
      (world, { unitroller, apiKey }) => verifyUnitroller(world, unitroller, apiKey.val)
    ),
    new Command<{ unitroller: Unitroller; pendingAdmin: AddressV }>(
      `
        #### AcceptAdmin

        * "AcceptAdmin" - Accept admin for this unitroller
          * E.g. "Unitroller AcceptAdmin"
      `,
      'AcceptAdmin',
      [new Arg('unitroller', getUnitroller, { implicit: true })],
      (world, from, { unitroller }) => acceptAdmin(world, from, unitroller)
    ),
    new Command<{ unitroller: Unitroller; pendingAdmin: AddressV }>(
      `
        #### SetPendingAdmin

        * "SetPendingAdmin admin:<Admin>" - Sets the pending admin for this unitroller
          * E.g. "Unitroller SetPendingAdmin Jared"
      `,
      'SetPendingAdmin',
      [new Arg('unitroller', getUnitroller, { implicit: true }), new Arg('pendingAdmin', getAddressV)],
      (world, from, { unitroller, pendingAdmin }) =>
        setPendingAdmin(world, from, unitroller, pendingAdmin.val)
    ),
    new Command<{ unitroller: Unitroller; comptrollerImpl: ComptrollerImpl }>(
      `
        #### SetPendingImpl

        * "SetPendingImpl impl:<Impl>" - Sets the pending comptroller implementation for this unitroller
          * E.g. "Unitroller SetPendingImpl MyScenImpl" - Sets the current comptroller implementation to MyScenImpl
      `,
      'SetPendingImpl',
      [
        new Arg('unitroller', getUnitroller, { implicit: true }),
        new Arg('comptrollerImpl', getComptrollerImpl)
      ],
      (world, from, { unitroller, comptrollerImpl }) =>
        setPendingImpl(world, from, unitroller, comptrollerImpl)
    )
  ];
}

export async function processUnitrollerEvent(
  world: World,
  event: Event,
  from: string | null
): Promise<World> {
  return await processCommandEvent<any>('Unitroller', unitrollerCommands(), world, event, from);
}

import { Event } from '../Event';
import { addAction, describeUser, World } from '../World';
import { ComptrollerImpl } from '../Contract/ComptrollerImpl';
import { Unitroller } from '../Contract/Unitroller';
import { invoke } from '../Invokation';
import { getAddressV, getArrayV, getEventV, getExpNumberV, getNumberV, getStringV, getCoreValue } from '../CoreValue';
import { ArrayV, AddressV, EventV, NumberV, StringV } from '../Value';
import { Arg, Command, View, processCommandEvent } from '../Command';
import { buildComptrollerImpl } from '../Builder/ComptrollerImplBuilder';
import { ComptrollerErrorReporter } from '../ErrorReporter';
import { getComptrollerImpl, getComptrollerImplData, getUnitroller } from '../ContractLookup';
import { verify } from '../Verify';
import { mergeContractABI } from '../Networks';
import { encodedNumber } from '../Encoding';
import { encodeABI } from '../Utils';

async function genComptrollerImpl(world: World, from: string, params: Event): Promise<World> {
  let { world: nextWorld, comptrollerImpl, comptrollerImplData } = await buildComptrollerImpl(
    world,
    from,
    params
  );
  world = nextWorld;

  world = addAction(
    world,
    `Added Comptroller Implementation (${comptrollerImplData.description}) at address ${comptrollerImpl._address}`,
    comptrollerImplData.invokation
  );

  return world;
}

async function mergeABI(
  world: World,
  from: string,
  comptrollerImpl: ComptrollerImpl,
  unitroller: Unitroller
): Promise<World> {
  if (!world.dryRun) {
    // Skip this specifically on dry runs since it's likely to crash due to a number of reasons
    world = await mergeContractABI(world, 'Comptroller', unitroller, unitroller.name, comptrollerImpl.name);
  }

  return world;
}

async function becomeG1(
  world: World,
  from: string,
  comptrollerImpl: ComptrollerImpl,
  unitroller: Unitroller,
  priceOracleAddr: string,
  closeFactor: encodedNumber,
  maxAssets: encodedNumber
): Promise<World> {
  let invokation = await invoke(
    world,
    comptrollerImpl.methods._become(unitroller._address, priceOracleAddr, closeFactor, maxAssets, false),
    from,
    ComptrollerErrorReporter
  );
  if (!world.dryRun) {
    // Skip this specifically on dry runs since it's likely to crash due to a number of reasons
    world = await mergeContractABI(world, 'Comptroller', unitroller, unitroller.name, comptrollerImpl.name);
  }

  world = addAction(
    world,
    `Become ${unitroller._address}'s Comptroller Impl with priceOracle=${priceOracleAddr},closeFactor=${closeFactor},maxAssets=${maxAssets}`,
    invokation
  );

  return world;
}

// Recome calls `become` on the G1 Comptroller, but passes a flag to not modify any of the initialization variables.
async function recome(
  world: World,
  from: string,
  comptrollerImpl: ComptrollerImpl,
  unitroller: Unitroller
): Promise<World> {
  let invokation = await invoke(
    world,
    comptrollerImpl.methods._become(
      unitroller._address,
      '0x0000000000000000000000000000000000000000',
      0,
      0,
      true
    ),
    from,
    ComptrollerErrorReporter
  );

  world = await mergeContractABI(world, 'Comptroller', unitroller, unitroller.name, comptrollerImpl.name);

  world = addAction(world, `Recome ${unitroller._address}'s Comptroller Impl`, invokation);

  return world;
}

async function becomeG2(
  world: World,
  from: string,
  comptrollerImpl: ComptrollerImpl,
  unitroller: Unitroller
): Promise<World> {
  let invokation = await invoke(
    world,
    comptrollerImpl.methods._become(unitroller._address),
    from,
    ComptrollerErrorReporter
  );

  if (!world.dryRun) {
    // Skip this specifically on dry runs since it's likely to crash due to a number of reasons
    world = await mergeContractABI(world, 'Comptroller', unitroller, unitroller.name, comptrollerImpl.name);
  }

  world = addAction(world, `Become ${unitroller._address}'s Comptroller Impl`, invokation);

  return world;
}

async function becomeG3(
  world: World,
  from: string,
  comptrollerImpl: ComptrollerImpl,
  unitroller: Unitroller,
  compRate: encodedNumber,
  compMarkets: string[],
  otherMarkets: string[]
): Promise<World> {
  let invokation = await invoke(
    world,
    comptrollerImpl.methods._become(unitroller._address, compRate, compMarkets, otherMarkets),
    from,
    ComptrollerErrorReporter
  );

  if (!world.dryRun) {
    // Skip this specifically on dry runs since it's likely to crash due to a number of reasons
    world = await mergeContractABI(world, 'Comptroller', unitroller, unitroller.name, comptrollerImpl.name);
  }

  world = addAction(world, `Become ${unitroller._address}'s Comptroller Impl`, invokation);

  return world;
}

async function becomeG4(
  world: World,
  from: string,
  comptrollerImpl: ComptrollerImpl,
  unitroller: Unitroller
): Promise<World> {
  let invokation = await invoke(
    world,
    comptrollerImpl.methods._become(unitroller._address),
    from,
    ComptrollerErrorReporter
  );

  if (!world.dryRun) {
    // Skip this specifically on dry runs since it's likely to crash due to a number of reasons
    world = await mergeContractABI(world, 'Comptroller', unitroller, unitroller.name, comptrollerImpl.name);
  }

  world = addAction(world, `Become ${unitroller._address}'s Comptroller Impl`, invokation);

  return world;
}

async function becomeG5(
  world: World,
  from: string,
  comptrollerImpl: ComptrollerImpl,
  unitroller: Unitroller
): Promise<World> {
  let invokation = await invoke(
    world,
    comptrollerImpl.methods._become(unitroller._address),
    from,
    ComptrollerErrorReporter
  );

  if (!world.dryRun) {
    // Skip this specifically on dry runs since it's likely to crash due to a number of reasons
    world = await mergeContractABI(world, 'Comptroller', unitroller, unitroller.name, comptrollerImpl.name);
  }

  world = addAction(world, `Become ${unitroller._address}'s Comptroller Impl`, invokation);

  return world;
}

async function becomeG6(
  world: World,
  from: string,
  comptrollerImpl: ComptrollerImpl,
  unitroller: Unitroller
): Promise<World> {
  let invokation = await invoke(
    world,
    comptrollerImpl.methods._become(unitroller._address),
    from,
    ComptrollerErrorReporter
  );

  if (!world.dryRun) {
    // Skip this specifically on dry runs since it's likely to crash due to a number of reasons
    world = await mergeContractABI(world, 'Comptroller', unitroller, unitroller.name, comptrollerImpl.name);
  }

  world = addAction(world, `Become ${unitroller._address}'s Comptroller Impl`, invokation);

  return world;
}

async function become(
  world: World,
  from: string,
  comptrollerImpl: ComptrollerImpl,
  unitroller: Unitroller
): Promise<World> {
  let invokation = await invoke(
    world,
    comptrollerImpl.methods._become(unitroller._address),
    from,
    ComptrollerErrorReporter
  );

  if (!world.dryRun) {
    // Skip this specifically on dry runs since it's likely to crash due to a number of reasons
    world = await mergeContractABI(world, 'Comptroller', unitroller, unitroller.name, comptrollerImpl.name);
  }

  world = addAction(world, `Become ${unitroller._address}'s Comptroller Impl`, invokation);

  return world;
}

async function verifyComptrollerImpl(
  world: World,
  comptrollerImpl: ComptrollerImpl,
  name: string,
  contract: string,
  apiKey: string
): Promise<World> {
  if (world.isLocalNetwork()) {
    world.printer.printLine(`Politely declining to verify on local network: ${world.network}.`);
  } else {
    await verify(world, apiKey, name, contract, comptrollerImpl._address);
  }

  return world;
}

export function comptrollerImplCommands() {
  return [
    new Command<{ comptrollerImplParams: EventV }>(
      `
        #### Deploy

        * "ComptrollerImpl Deploy ...comptrollerImplParams" - Generates a new Comptroller Implementation
          * E.g. "ComptrollerImpl Deploy MyScen Scenario"
      `,
      'Deploy',
      [new Arg('comptrollerImplParams', getEventV, { variadic: true })],
      (world, from, { comptrollerImplParams }) => genComptrollerImpl(world, from, comptrollerImplParams.val)
    ),
    new View<{ comptrollerImplArg: StringV; apiKey: StringV }>(
      `
        #### Verify

        * "ComptrollerImpl <Impl> Verify apiKey:<String>" - Verifies Comptroller Implemetation in Etherscan
          * E.g. "ComptrollerImpl Verify "myApiKey"
      `,
      'Verify',
      [new Arg('comptrollerImplArg', getStringV), new Arg('apiKey', getStringV)],
      async (world, { comptrollerImplArg, apiKey }) => {
        let [comptrollerImpl, name, data] = await getComptrollerImplData(world, comptrollerImplArg.val);

        return await verifyComptrollerImpl(world, comptrollerImpl, name, data.get('contract')!, apiKey.val);
      },
      { namePos: 1 }
    ),
    new Command<{
      unitroller: Unitroller;
      comptrollerImpl: ComptrollerImpl;
      priceOracle: AddressV;
      closeFactor: NumberV;
      maxAssets: NumberV;
    }>(
      `
        #### BecomeG1

        * "ComptrollerImpl <Impl> BecomeG1 priceOracle:<Number> closeFactor:<Exp> maxAssets:<Number>" - Become the comptroller, if possible.
          * E.g. "ComptrollerImpl MyImpl BecomeG1
      `,
      'BecomeG1',
      [
        new Arg('unitroller', getUnitroller, { implicit: true }),
        new Arg('comptrollerImpl', getComptrollerImpl),
        new Arg('priceOracle', getAddressV),
        new Arg('closeFactor', getExpNumberV),
        new Arg('maxAssets', getNumberV)
      ],
      (world, from, { unitroller, comptrollerImpl, priceOracle, closeFactor, maxAssets }) =>
        becomeG1(
          world,
          from,
          comptrollerImpl,
          unitroller,
          priceOracle.val,
          closeFactor.encode(),
          maxAssets.encode()
        ),
      { namePos: 1 }
    ),

    new Command<{
      unitroller: Unitroller;
      comptrollerImpl: ComptrollerImpl;
    }>(
      `
        #### BecomeG2

        * "ComptrollerImpl <Impl> BecomeG2" - Become the comptroller, if possible.
          * E.g. "ComptrollerImpl MyImpl BecomeG2
      `,
      'BecomeG2',
      [
        new Arg('unitroller', getUnitroller, { implicit: true }),
        new Arg('comptrollerImpl', getComptrollerImpl)
      ],
      (world, from, { unitroller, comptrollerImpl }) => becomeG2(world, from, comptrollerImpl, unitroller),
      { namePos: 1 }
    ),

    new Command<{
      unitroller: Unitroller;
      comptrollerImpl: ComptrollerImpl;
      compRate: NumberV;
      compMarkets: ArrayV<AddressV>;
      otherMarkets: ArrayV<AddressV>;
    }>(
      `
        #### BecomeG3

        * "ComptrollerImpl <Impl> BecomeG3 <Rate> <CompMarkets> <OtherMarkets>" - Become the comptroller, if possible.
          * E.g. "ComptrollerImpl MyImpl BecomeG3 0.1e18 [cDAI, cETH, cUSDC]
      `,
      'BecomeG3',
      [
        new Arg('unitroller', getUnitroller, { implicit: true }),
        new Arg('comptrollerImpl', getComptrollerImpl),
        new Arg('compRate', getNumberV, { default: new NumberV(1e18) }),
        new Arg('compMarkets', getArrayV(getAddressV),  {default: new ArrayV([]) }),
        new Arg('otherMarkets', getArrayV(getAddressV), { default: new ArrayV([]) })
      ],
      (world, from, { unitroller, comptrollerImpl, compRate, compMarkets, otherMarkets }) => {
        return becomeG3(world, from, comptrollerImpl, unitroller, compRate.encode(), compMarkets.val.map(a => a.val), otherMarkets.val.map(a => a.val))
      },
      { namePos: 1 }
    ),
  
    new Command<{
      unitroller: Unitroller;
      comptrollerImpl: ComptrollerImpl;
    }>(
      `
        #### BecomeG4
        * "ComptrollerImpl <Impl> BecomeG4" - Become the comptroller, if possible.
          * E.g. "ComptrollerImpl MyImpl BecomeG4
      `,
      'BecomeG4',
      [
        new Arg('unitroller', getUnitroller, { implicit: true }),
        new Arg('comptrollerImpl', getComptrollerImpl)
      ],
      (world, from, { unitroller, comptrollerImpl }) => {
        return becomeG4(world, from, comptrollerImpl, unitroller)
      },
      { namePos: 1 }
    ),

    new Command<{
      unitroller: Unitroller;
      comptrollerImpl: ComptrollerImpl;
    }>(
      `
        #### BecomeG5
        * "ComptrollerImpl <Impl> BecomeG5" - Become the comptroller, if possible.
          * E.g. "ComptrollerImpl MyImpl BecomeG5
      `,
      'BecomeG5',
      [
        new Arg('unitroller', getUnitroller, { implicit: true }),
        new Arg('comptrollerImpl', getComptrollerImpl)
      ],
      (world, from, { unitroller, comptrollerImpl }) => {
        return becomeG5(world, from, comptrollerImpl, unitroller)
      },
      { namePos: 1 }
    ),

    new Command<{
      unitroller: Unitroller;
      comptrollerImpl: ComptrollerImpl;
    }>(
      `
        #### BecomeG6
        * "ComptrollerImpl <Impl> BecomeG6" - Become the comptroller, if possible.
          * E.g. "ComptrollerImpl MyImpl BecomeG6
      `,
      'BecomeG6',
      [
        new Arg('unitroller', getUnitroller, { implicit: true }),
        new Arg('comptrollerImpl', getComptrollerImpl)
      ],
      (world, from, { unitroller, comptrollerImpl }) => {
        return becomeG6(world, from, comptrollerImpl, unitroller)
      },
      { namePos: 1 }
    ),

    new Command<{
      unitroller: Unitroller;
      comptrollerImpl: ComptrollerImpl;
    }>(
      `
        #### Become

        * "ComptrollerImpl <Impl> Become <Rate> <CompMarkets> <OtherMarkets>" - Become the comptroller, if possible.
          * E.g. "ComptrollerImpl MyImpl Become 0.1e18 [cDAI, cETH, cUSDC]
      `,
      'Become',
      [
        new Arg('unitroller', getUnitroller, { implicit: true }),
        new Arg('comptrollerImpl', getComptrollerImpl)
      ],
      (world, from, { unitroller, comptrollerImpl }) => {
        return become(world, from, comptrollerImpl, unitroller)
      },
      { namePos: 1 }
    ),

    new Command<{
      unitroller: Unitroller;
      comptrollerImpl: ComptrollerImpl;
    }>(
      `
        #### MergeABI

        * "ComptrollerImpl <Impl> MergeABI" - Merges the ABI, as if it was a become.
          * E.g. "ComptrollerImpl MyImpl MergeABI
      `,
      'MergeABI',
      [
        new Arg('unitroller', getUnitroller, { implicit: true }),
        new Arg('comptrollerImpl', getComptrollerImpl)
      ],
      (world, from, { unitroller, comptrollerImpl }) => mergeABI(world, from, comptrollerImpl, unitroller),
      { namePos: 1 }
    ),
    new Command<{ unitroller: Unitroller; comptrollerImpl: ComptrollerImpl }>(
      `
        #### Recome

        * "ComptrollerImpl <Impl> Recome" - Recome the comptroller
          * E.g. "ComptrollerImpl MyImpl Recome
      `,
      'Recome',
      [
        new Arg('unitroller', getUnitroller, { implicit: true }),
        new Arg('comptrollerImpl', getComptrollerImpl)
      ],
      (world, from, { unitroller, comptrollerImpl }) => recome(world, from, comptrollerImpl, unitroller),
      { namePos: 1 }
    )
  ];
}

export async function processComptrollerImplEvent(
  world: World,
  event: Event,
  from: string | null
): Promise<World> {
  return await processCommandEvent<any>('ComptrollerImpl', comptrollerImplCommands(), world, event, from);
}

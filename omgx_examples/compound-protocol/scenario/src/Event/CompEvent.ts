import { Event } from '../Event';
import { addAction, World, describeUser } from '../World';
import { Comp, CompScenario } from '../Contract/Comp';
import { buildComp } from '../Builder/CompBuilder';
import { invoke } from '../Invokation';
import {
  getAddressV,
  getEventV,
  getNumberV,
  getStringV,
} from '../CoreValue';
import {
  AddressV,
  EventV,
  NumberV,
  StringV
} from '../Value';
import { Arg, Command, processCommandEvent, View } from '../Command';
import { getComp } from '../ContractLookup';
import { NoErrorReporter } from '../ErrorReporter';
import { verify } from '../Verify';
import { encodedNumber } from '../Encoding';

async function genComp(world: World, from: string, params: Event): Promise<World> {
  let { world: nextWorld, comp, tokenData } = await buildComp(world, from, params);
  world = nextWorld;

  world = addAction(
    world,
    `Deployed Comp (${comp.name}) to address ${comp._address}`,
    tokenData.invokation
  );

  return world;
}

async function verifyComp(world: World, comp: Comp, apiKey: string, modelName: string, contractName: string): Promise<World> {
  if (world.isLocalNetwork()) {
    world.printer.printLine(`Politely declining to verify on local network: ${world.network}.`);
  } else {
    await verify(world, apiKey, modelName, contractName, comp._address);
  }

  return world;
}

async function approve(world: World, from: string, comp: Comp, address: string, amount: NumberV): Promise<World> {
  let invokation = await invoke(world, comp.methods.approve(address, amount.encode()), from, NoErrorReporter);

  world = addAction(
    world,
    `Approved Comp token for ${from} of ${amount.show()}`,
    invokation
  );

  return world;
}

async function transfer(world: World, from: string, comp: Comp, address: string, amount: NumberV): Promise<World> {
  let invokation = await invoke(world, comp.methods.transfer(address, amount.encode()), from, NoErrorReporter);

  world = addAction(
    world,
    `Transferred ${amount.show()} Comp tokens from ${from} to ${address}`,
    invokation
  );

  return world;
}

async function transferFrom(world: World, from: string, comp: Comp, owner: string, spender: string, amount: NumberV): Promise<World> {
  let invokation = await invoke(world, comp.methods.transferFrom(owner, spender, amount.encode()), from, NoErrorReporter);

  world = addAction(
    world,
    `"Transferred from" ${amount.show()} Comp tokens from ${owner} to ${spender}`,
    invokation
  );

  return world;
}

async function transferScenario(world: World, from: string, comp: CompScenario, addresses: string[], amount: NumberV): Promise<World> {
  let invokation = await invoke(world, comp.methods.transferScenario(addresses, amount.encode()), from, NoErrorReporter);

  world = addAction(
    world,
    `Transferred ${amount.show()} Comp tokens from ${from} to ${addresses}`,
    invokation
  );

  return world;
}

async function transferFromScenario(world: World, from: string, comp: CompScenario, addresses: string[], amount: NumberV): Promise<World> {
  let invokation = await invoke(world, comp.methods.transferFromScenario(addresses, amount.encode()), from, NoErrorReporter);

  world = addAction(
    world,
    `Transferred ${amount.show()} Comp tokens from ${addresses} to ${from}`,
    invokation
  );

  return world;
}

async function delegate(world: World, from: string, comp: Comp, account: string): Promise<World> {
  let invokation = await invoke(world, comp.methods.delegate(account), from, NoErrorReporter);

  world = addAction(
    world,
    `"Delegated from" ${from} to ${account}`,
    invokation
  );

  return world;
}

async function setBlockNumber(
  world: World,
  from: string,
  comp: Comp,
  blockNumber: NumberV
): Promise<World> {
  return addAction(
    world,
    `Set Comp blockNumber to ${blockNumber.show()}`,
    await invoke(world, comp.methods.setBlockNumber(blockNumber.encode()), from)
  );
}

export function compCommands() {
  return [
    new Command<{ params: EventV }>(`
        #### Deploy

        * "Deploy ...params" - Generates a new Comp token
          * E.g. "Comp Deploy"
      `,
      "Deploy",
      [
        new Arg("params", getEventV, { variadic: true })
      ],
      (world, from, { params }) => genComp(world, from, params.val)
    ),

    new View<{ comp: Comp, apiKey: StringV, contractName: StringV }>(`
        #### Verify

        * "<Comp> Verify apiKey:<String> contractName:<String>=Comp" - Verifies Comp token in Etherscan
          * E.g. "Comp Verify "myApiKey"
      `,
      "Verify",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("apiKey", getStringV),
        new Arg("contractName", getStringV, { default: new StringV("Comp") })
      ],
      async (world, { comp, apiKey, contractName }) => {
        return await verifyComp(world, comp, apiKey.val, comp.name, contractName.val)
      }
    ),

    new Command<{ comp: Comp, spender: AddressV, amount: NumberV }>(`
        #### Approve

        * "Comp Approve spender:<Address> <Amount>" - Adds an allowance between user and address
          * E.g. "Comp Approve Geoff 1.0e18"
      `,
      "Approve",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("spender", getAddressV),
        new Arg("amount", getNumberV)
      ],
      (world, from, { comp, spender, amount }) => {
        return approve(world, from, comp, spender.val, amount)
      }
    ),

    new Command<{ comp: Comp, recipient: AddressV, amount: NumberV }>(`
        #### Transfer

        * "Comp Transfer recipient:<User> <Amount>" - Transfers a number of tokens via "transfer" as given user to recipient (this does not depend on allowance)
          * E.g. "Comp Transfer Torrey 1.0e18"
      `,
      "Transfer",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("recipient", getAddressV),
        new Arg("amount", getNumberV)
      ],
      (world, from, { comp, recipient, amount }) => transfer(world, from, comp, recipient.val, amount)
    ),

    new Command<{ comp: Comp, owner: AddressV, spender: AddressV, amount: NumberV }>(`
        #### TransferFrom

        * "Comp TransferFrom owner:<User> spender:<User> <Amount>" - Transfers a number of tokens via "transfeFrom" to recipient (this depends on allowances)
          * E.g. "Comp TransferFrom Geoff Torrey 1.0e18"
      `,
      "TransferFrom",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("owner", getAddressV),
        new Arg("spender", getAddressV),
        new Arg("amount", getNumberV)
      ],
      (world, from, { comp, owner, spender, amount }) => transferFrom(world, from, comp, owner.val, spender.val, amount)
    ),

    new Command<{ comp: CompScenario, recipients: AddressV[], amount: NumberV }>(`
        #### TransferScenario

        * "Comp TransferScenario recipients:<User[]> <Amount>" - Transfers a number of tokens via "transfer" to the given recipients (this does not depend on allowance)
          * E.g. "Comp TransferScenario (Jared Torrey) 10"
      `,
      "TransferScenario",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("recipients", getAddressV, { mapped: true }),
        new Arg("amount", getNumberV)
      ],
      (world, from, { comp, recipients, amount }) => transferScenario(world, from, comp, recipients.map(recipient => recipient.val), amount)
    ),

    new Command<{ comp: CompScenario, froms: AddressV[], amount: NumberV }>(`
        #### TransferFromScenario

        * "Comp TransferFromScenario froms:<User[]> <Amount>" - Transfers a number of tokens via "transferFrom" from the given users to msg.sender (this depends on allowance)
          * E.g. "Comp TransferFromScenario (Jared Torrey) 10"
      `,
      "TransferFromScenario",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("froms", getAddressV, { mapped: true }),
        new Arg("amount", getNumberV)
      ],
      (world, from, { comp, froms, amount }) => transferFromScenario(world, from, comp, froms.map(_from => _from.val), amount)
    ),

    new Command<{ comp: Comp, account: AddressV }>(`
        #### Delegate

        * "Comp Delegate account:<Address>" - Delegates votes to a given account
          * E.g. "Comp Delegate Torrey"
      `,
      "Delegate",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("account", getAddressV),
      ],
      (world, from, { comp, account }) => delegate(world, from, comp, account.val)
    ),
    new Command<{ comp: Comp, blockNumber: NumberV }>(`
      #### SetBlockNumber

      * "SetBlockNumber <Seconds>" - Sets the blockTimestamp of the Comp Harness
      * E.g. "Comp SetBlockNumber 500"
      `,
        'SetBlockNumber',
        [new Arg('comp', getComp, { implicit: true }), new Arg('blockNumber', getNumberV)],
        (world, from, { comp, blockNumber }) => setBlockNumber(world, from, comp, blockNumber)
      )
  ];
}

export async function processCompEvent(world: World, event: Event, from: string | null): Promise<World> {
  return await processCommandEvent<any>("Comp", compCommands(), world, event, from);
}

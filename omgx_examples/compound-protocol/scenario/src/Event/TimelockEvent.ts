import { Event } from '../Event';
import { addAction, World } from '../World';
import { Timelock } from '../Contract/Timelock';
import { buildTimelock, TimelockData } from '../Builder/TimelockBuilder';
import { invoke } from '../Invokation';
import { getAddressV, getEventV, getNumberV, getStringV, getCoreValue } from '../CoreValue';
import { AddressV, EventV, NumberV, StringV } from '../Value';
import { Arg, Command, processCommandEvent, View } from '../Command';
import { getTimelock } from '../ContractLookup';
import { verify } from '../Verify';
import { decodeParameters, encodeParameters } from '../Utils';

async function genTimelock(world: World, from: string, params: Event): Promise<World> {
  let { world: nextWorld, timelock, timelockData } = await buildTimelock(world, from, params);
  world = nextWorld;

  world = addAction(world, `Deployed Timelock to address ${timelock._address}`, timelockData.invokation);

  return world;
}

async function acceptAdmin(world: World, from: string, timeLock: Timelock): Promise<World> {
  return addAction(
    world,
    `Set Timelock admin to ${from}`,
    await invoke(world, timeLock.methods.acceptAdmin(), from)
  );
}

async function setPendingAdmin(
  world: World,
  from: string,
  timeLock: Timelock,
  admin: string
): Promise<World> {
  return addAction(
    world,
    `Set Timelock admin to ${admin}`,
    await invoke(world, timeLock.methods.setPendingAdmin(admin), from)
  );
}

async function setAdmin(
  world: World,
  from: string,
  timeLock: Timelock,
  admin: string
): Promise<World> {
  return addAction(
    world,
    `Set Timelock admin to ${admin}`,
    await invoke(world, timeLock.methods.harnessSetAdmin(admin), from)
  );
}

async function setDelay(world: World, from: string, timeLock: Timelock, delay: NumberV): Promise<World> {
  return addAction(
    world,
    `Set Timelock delay to ${delay.show()}`,
    await invoke(world, timeLock.methods.setDelay(delay.encode()), from)
  );
}

async function harnessFastForward(
  world: World,
  from: string,
  timeLock: Timelock,
  seconds: NumberV
): Promise<World> {
  return addAction(
    world,
    `Set Timelock blockTimestamp forward by ${seconds.show()}`,
    await invoke(world, timeLock.methods.harnessFastForward(seconds.encode()), from)
  );
}

async function harnessSetBlockTimestamp(
  world: World,
  from: string,
  timeLock: Timelock,
  seconds: NumberV
): Promise<World> {
  return addAction(
    world,
    `Set Timelock blockTimestamp to ${seconds.show()}`,
    await invoke(world, timeLock.methods.harnessSetBlockTimestamp(seconds.encode()), from)
  );
}

async function queueTransaction(
  world: World,
  from: string,
  timeLock: Timelock,
  target: string,
  value: NumberV,
  signature: string,
  data: string,
  eta: NumberV
): Promise<World> {
  const dataArgs = decodeParameters(world, signature, data);
  const etaString = eta.show();
  const dateFromEta = new Date(Number(etaString) * 1000);

  return addAction(
    world,
    `Queue transaction on Timelock with target: ${target}\nvalue: ${value.show()}\nsignature: ${signature}\ndata: ${data} (args: ${dataArgs.join(
      ', '
    )})\neta: ${etaString} (${dateFromEta.toString()})`,
    await invoke(
      world,
      timeLock.methods.queueTransaction(target, value.encode(), signature, data, eta.encode()),
      from
    )
  );
}

async function cancelTransaction(
  world: World,
  from: string,
  timeLock: Timelock,
  target: string,
  value: NumberV,
  signature: string,
  data: string,
  eta: NumberV
): Promise<World> {
  return addAction(
    world,
    `Cancel transaction on Timelock with target: ${target} value: ${value.show()} signature: ${signature} data: ${data} eta: ${eta.show()}`,
    await invoke(
      world,
      timeLock.methods.cancelTransaction(target, value.encode(), signature, data, eta.encode()),
      from
    )
  );
}

async function executeTransaction(
  world: World,
  from: string,
  timeLock: Timelock,
  target: string,
  value: NumberV,
  signature: string,
  data: string,
  eta: NumberV
): Promise<World> {
  const dataArgs = decodeParameters(world, signature, data);
  const etaString = eta.show();
  const dateFromEta = new Date(Number(etaString) * 1000);

  return addAction(
    world,
    `Execute transaction on Timelock with target: ${target}\nvalue: ${value.show()}\nsignature: ${signature}\ndata: ${data} (args: ${dataArgs.join(
      ', '
    )})\neta: ${etaString} (${dateFromEta.toString()})`,
    await invoke(
      world,
      timeLock.methods.executeTransaction(target, value.encode(), signature, data, eta.encode()),
      from
    )
  );
}

async function verifyTimelock(
  world: World,
  timelock: Timelock,
  apiKey: string,
  contractName: string
): Promise<World> {
  if (world.isLocalNetwork()) {
    world.printer.printLine(`Politely declining to verify on local network: ${world.network}.`);
  } else {
    await verify(world, apiKey, 'Timelock', contractName, timelock._address);
  }

  return world;
}

export function timelockCommands() {
  return [
    new Command<{ params: EventV }>(
      `
        #### Deploy

        * "Deploy ...params" - Generates a new price oracle proxy
          * E.g. "Timelock Deploy Geoff 604800"
      `,
      'Deploy',
      [new Arg('params', getEventV, { variadic: true })],
      (world, from, { params }) => genTimelock(world, from, params.val)
    ),
    new Command<{ timelock: Timelock; seconds: NumberV }>(
      `
        #### FastForward

        * "FastForward <Seconds>" - Sets the blockTimestamp of the TimelockHarness forward
        * E.g. "Timelock FastForward 604800"
    `,
      'FastForward',
      [new Arg('timelock', getTimelock, { implicit: true }), new Arg('seconds', getNumberV)],
      (world, from, { timelock, seconds }) => harnessFastForward(world, from, timelock, seconds)
    ),
    new Command<{ timelock: Timelock; seconds: NumberV }>(
      `
        #### SetBlockTimestamp

        * "SetBlockTimestamp <Seconds>" - Sets the blockTimestamp of the TimelockHarness
        * E.g. "Timelock SetBlockTimestamp 1569973599"
    `,
      'SetBlockTimestamp',
      [new Arg('timelock', getTimelock, { implicit: true }), new Arg('seconds', getNumberV)],
      (world, from, { timelock, seconds }) => harnessSetBlockTimestamp(world, from, timelock, seconds)
    ),
    new Command<{ timelock: Timelock; delay: NumberV }>(
      `
        #### SetDelay

        * "SetDelay <Delay>" - Sets the delay for the Timelock
        * E.g. "Timelock SetDelay 604800"
    `,
      'SetDelay',
      [new Arg('timelock', getTimelock, { implicit: true }), new Arg('delay', getNumberV)],
      (world, from, { timelock, delay }) => setDelay(world, from, timelock, delay)
    ),
    new Command<{ timelock: Timelock }>(
      `
        #### AcceptAdmin

        * "AcceptAdmin" - Accept the admin for the Timelock
        * E.g. "Timelock AcceptAdmin"
    `,
      'AcceptAdmin',
      [new Arg('timelock', getTimelock, { implicit: true })],
      (world, from, { timelock }) => acceptAdmin(world, from, timelock)
    ),
    new Command<{ timelock: Timelock; admin: AddressV }>(
      `
        #### SetPendingAdmin

        * "SetPendingAdmin <Address>" - Sets the pending admin for the Timelock
        * E.g. "Timelock SetPendingAdmin \"0x0000000000000000000000000000000000000000\""
    `,
      'SetPendingAdmin',
      [new Arg('timelock', getTimelock, { implicit: true }), new Arg('admin', getAddressV)],
      (world, from, { timelock, admin }) => setPendingAdmin(world, from, timelock, admin.val)
    ),
    new Command<{ timelock: Timelock; admin: AddressV }>(
      `
        #### SetAdmin

        * "SetAdmin <Address>" - Sets the admin for the Timelock through the harness
        * E.g. "Timelock SetAdmin \"0x0000000000000000000000000000000000000000\""
    `,
      'SetAdmin',
      [new Arg('timelock', getTimelock, { implicit: true }), new Arg('admin', getAddressV)],
      (world, from, { timelock, admin }) => setAdmin(world, from, timelock, admin.val)
    ),
    new Command<{
      timelock: Timelock;
      target: AddressV;
      value: NumberV;
      eta: NumberV;
      signature: StringV;
      data: StringV[];
    }>(
      `
        #### QueueTransaction

        * "QueueTransaction target:<Address> value:<Number> eta:<Number> signature:<String> ...funArgs:<CoreValue>" - Queues a transaction for the Timelock
        * E.g. "Timelock QueueTransaction \"0x0000000000000000000000000000000000000000\" 0 1569286014 \"setDelay(uint256)\" 60680"
        *
    `,
      'QueueTransaction',
      [
        new Arg('timelock', getTimelock, { implicit: true }),
        new Arg('target', getAddressV),
        new Arg('value', getNumberV),
        new Arg('eta', getNumberV),
        new Arg('signature', getStringV),
        new Arg('data', getCoreValue, { variadic: true, mapped: true })
      ],
      (world, from, { timelock, target, value, signature, data, eta }) => {
        const encodedData = encodeParameters(world, signature.val, data.map(a => a.val));
        return queueTransaction(world, from, timelock, target.val, value, signature.val, encodedData, eta);
      }
    ),
    new Command<{
      timelock: Timelock;
      target: AddressV;
      value: NumberV;
      eta: NumberV;
      signature: StringV;
      data: StringV[];
    }>(
      `
        #### CancelTransaction

        * "CancelTransaction target:<Address> value:<Number> eta:<Number> signature:<String> ...funArgs:<CoreValue>" - Cancels a transaction from the Timelock
        * E.g. "Timelock CancelTransaction \"0x0000000000000000000000000000000000000000\" 0 1569286014 \"setDelay(uint256)\" 60680"
    `,
      'CancelTransaction',
      [
        new Arg('timelock', getTimelock, { implicit: true }),
        new Arg('target', getAddressV),
        new Arg('value', getNumberV),
        new Arg('eta', getNumberV),
        new Arg('signature', getStringV),
        new Arg('data', getCoreValue, { variadic: true, mapped: true })
      ],
      (world, from, { timelock, target, value, signature, data, eta }) => {
        const encodedData = encodeParameters(world, signature.val, data.map(a => a.val));
        return cancelTransaction(world, from, timelock, target.val, value, signature.val, encodedData, eta);
      }
    ),
    new Command<{
      timelock: Timelock;
      target: AddressV;
      value: NumberV;
      eta: NumberV;
      signature: StringV;
      data: StringV[];
    }>(
      `
        #### ExecuteTransaction

        * "ExecuteTransaction target:<Address> value:<Number> eta:<Number> signature:<String> ...funArgs:<CoreValue>" - Executes a transaction from the Timelock
        * E.g. "Timelock ExecuteTransaction \"0x0000000000000000000000000000000000000000\" 0 1569286014 \"setDelay(uint256)\" 60680"
    `,
      'ExecuteTransaction',
      [
        new Arg('timelock', getTimelock, { implicit: true }),
        new Arg('target', getAddressV),
        new Arg('value', getNumberV),
        new Arg('eta', getNumberV),
        new Arg('signature', getStringV),
        new Arg('data', getCoreValue, { variadic: true, mapped: true })
      ],
      (world, from, { timelock, target, value, signature, data, eta }) => {
        const encodedData = encodeParameters(world, signature.val, data.map(a => a.val));
        return executeTransaction(world, from, timelock, target.val, value, signature.val, encodedData, eta);
      }
    ),
    new View<{ timelock: Timelock; apiKey: StringV; contractName: StringV }>(
      `
        #### Verify

        * "Verify apiKey:<String> contractName:<String>=Timelock" - Verifies Timelock in Etherscan
          * E.g. "Timelock Verify "myApiKey"
      `,
      'Verify',
      [
        new Arg('timelock', getTimelock, { implicit: true }),
        new Arg('apiKey', getStringV),
        new Arg('contractName', getStringV, { default: new StringV('Timelock') })
      ],
      (world, { timelock, apiKey, contractName }) =>
        verifyTimelock(world, timelock, apiKey.val, contractName.val)
    )
  ];
}

export async function processTimelockEvent(world: World, event: Event, from: string | null): Promise<World> {
  return await processCommandEvent<any>('Timelock', timelockCommands(), world, event, from);
}

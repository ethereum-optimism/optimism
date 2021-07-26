import { Expect, throwExpect } from './Assert';
import { Action } from './Action';
import { Contract } from './Contract';
import { Record } from 'immutable';
import { Printer } from './Printer';
import { Invariant } from './Invariant';
import { SuccessInvariant } from './Invariant/SuccessInvariant';
import { RemainsInvariant } from './Invariant/RemainsInvariant';
import { StaticInvariant } from './Invariant/StaticInvariant';
import { Expectation } from './Expectation';
import { formatResult } from './ErrorReporter';
import { Invokation, InvokationOpts } from './Invokation';
import { Event } from './Event';
import { formatEvent } from './Formatter';
import { Map } from 'immutable';
import { Settings } from './Settings';
import { Accounts, loadAccounts } from './Accounts';
import Web3 from 'web3';
import { Saddle } from 'eth-saddle';
import { Command, Fetcher } from './Command';
import { Value} from './Value';

const startingBlockNumber = 1000;

type ContractIndex = { [address: string]: Contract };
type Counter = { value: number };
type EventDecoder = { [eventSignature: string]: (log: any) => any };

export interface WorldProps {
  actions: Action<any>[];
  event: Event | null;
  lastInvokation: Invokation<any> | null;
  newInvokation: boolean;
  blockNumber: number;
  gasCounter: Counter;
  lastContract: Contract | null;
  invariants: Invariant[];
  expectations: Expectation[];
  contractIndex: ContractIndex;
  contractData: Map<string, object>;
  expect: Expect;
  web3: Web3 | null;
  saddle: Saddle | null;
  printer: Printer | null;
  network: string | null;
  dryRun: boolean;
  verbose: boolean;
  settings: Settings;
  accounts: Accounts | null;
  invokationOpts: InvokationOpts;
  trxInvokationOpts: Map<string, any>;
  basePath: string | null;
  totalGas: number | null;
  eventDecoder: EventDecoder;
  fs: object | null;
  commands: Command<any>[] | undefined;
  fetchers: Fetcher<any, Value>[] | undefined;
}

const defaultWorldProps: WorldProps = {
  actions: <Action<any>[]>[],
  event: null,
  lastInvokation: null,
  newInvokation: false,
  blockNumber: 0,
  gasCounter: {value: 0},
  lastContract: null,
  invariants: [],
  expectations: [],
  contractIndex: {},
  contractData: Map({}),
  expect: throwExpect,
  web3: null,
  saddle: null,
  printer: null,
  network: null,
  dryRun: false,
  verbose: false,
  settings: Settings.default(null, null),
  accounts: null,
  invokationOpts: {},
  trxInvokationOpts: Map({}),
  basePath: null,
  totalGas: null,
  eventDecoder: {},
  fs: null,
  commands: undefined,
  fetchers: undefined,
};

export class World extends Record(defaultWorldProps) {
  public readonly actions!: Action<any>[];
  public readonly event!: Event | null;
  public readonly value!: number | null;
  public readonly lastInvokation!: Invokation<any> | null;
  public readonly newInvokation!: boolean;
  public readonly blockNumber!: number;
  public readonly gasCounter!: Counter;
  public readonly lastContract!: Contract | null;
  public readonly invariants!: Invariant[];
  public readonly expectations!: Expectation[];
  public readonly contractIndex!: ContractIndex;
  public readonly contractData!: Map<string, object>;
  public readonly expect!: Expect;
  public readonly web3!: Web3;
  public readonly saddle!: Saddle;
  public readonly printer!: Printer;
  public readonly network!: string;
  public readonly dryRun!: boolean;
  public readonly verbose!: boolean;
  public readonly settings!: Settings;
  public readonly accounts!: Accounts;
  public readonly invokationOpts!: InvokationOpts;
  public readonly trxInvokationOpts!: Map<string, any>;
  public readonly basePath!: string | null;

  public constructor(values?: Partial<WorldProps>) {
    values ? super(values) : super();
  }

  getInvokationOpts(baseOpts: InvokationOpts): InvokationOpts {
    return {
      ...baseOpts,
      ...this.invokationOpts,
      ...this.value ? {value: this.value.toString()} : {}
    };
  }

  isLocalNetwork(): boolean {
    return this.network === 'test' || this.network === 'development' || this.network === 'coverage';
  }

  async updateSettings(fn: (settings: Settings) => Promise<Settings>): Promise<World> {
    // TODO: Should we do an immutable update?
    const newSettings = await fn(this.settings);

    // TODO: Should we await or just let it clobber?
    await newSettings.save();

    return this.set('settings', newSettings);
  }

  defaultFrom(): string | null {
    let settingsFrom = this.settings.findAlias('Me');
    if (settingsFrom) {
      return settingsFrom;
    }

    let accountsDefault = this.accounts.get('default');
    if (accountsDefault) {
      return accountsDefault.address;
    }

    return null;
  }
}

export function loadInvokationOpts(world: World): World {
  let networkOpts = {};
  const networkOptsStr = process.env[`${world.network}_opts`];
  if (networkOptsStr) {
    networkOpts = JSON.parse(networkOptsStr);
  }

  return world.set('invokationOpts', networkOpts);
}

export function loadVerbose(world: World): World {
  return world.set('verbose', !!process.env['verbose']);
}

export function loadDryRun(world: World): World {
  return world.set('dryRun', !!process.env['dry_run']);
}

export async function loadSettings(world: World): Promise<World> {
  if (world.basePath) {
    return world.set('settings', await Settings.load(world.basePath, world.network));
  } else {
    return world;
  }
}

export async function initWorld(
  expect: Expect,
  printer: Printer,
  iweb3: Web3,
  saddle: Saddle,
  network: string,
  accounts: string[],
  basePath: string | null,
  totalGas: number | null
): Promise<World> {
  return new World({
    actions: [],
    event: null,
    lastInvokation: null,
    newInvokation: true,
    blockNumber: startingBlockNumber,
    gasCounter: {value: 0},
    lastContract: null,
    invariants: [new SuccessInvariant()], // Start with invariant success,
    expectations: [],
    contractIndex: {},
    contractData: Map({}),
    expect: expect,
    web3: iweb3,
    saddle: saddle,
    printer: printer,
    network: network,
    settings: Settings.default(basePath, null),
    accounts: loadAccounts(accounts),
    trxInvokationOpts: Map({}),
    basePath: basePath,
    totalGas: totalGas ? totalGas : null,
    eventDecoder: {},
    fs: network === 'test' ? {} : null
  });
}

export function setEvent(world: World, event: Event): World {
  return world.set('event', event);
}

export function addAction(world: World, log: string, invokation: Invokation<any>): World {
  const action = new Action(log, invokation);

  world = world.update('actions', actions => actions.concat([action]));

  // Print the action via the printer
  world.printer.printAction(action);

  return world.merge(world, {
    lastInvokation: invokation,
    newInvokation: true
  });
}

export function addInvariant(world: World, invariant: Invariant): World {
  return world.update('invariants', invariants => invariants.concat([invariant]));
}

export function addExpectation(world: World, expectation: Expectation): World {
  return world.update('expectations', expectations => expectations.concat([expectation]));
}

function getInvariantFilter(type: string) {
  let filters: { [filter: string]: (invariant: Invariant) => boolean } = {
    all: _invariant => true,
    success: invariant => !(invariant instanceof SuccessInvariant),
    remains: invariant => !(invariant instanceof RemainsInvariant),
    static: invariant => !(invariant instanceof StaticInvariant)
  };

  let filter = filters[type.toLowerCase()];

  if (!filter) {
    throw new Error(`Unknown invariant type \`${type}\` when wiping invariants.`);
  }

  return filter;
}

export function clearInvariants(world: World, type: string): World {
  let filter = getInvariantFilter(type);

  return world.update('invariants', invariants => world.invariants.filter(filter));
}

export function holdInvariants(world: World, type: string): World {
  let filter = getInvariantFilter(type);

  return world.update('invariants', invariants => {
    return world.invariants.map(invariant => {
      if (filter(invariant)) {
        invariant.held = true;
      }

      return invariant;
    });
  });
}

export async function checkExpectations(world: World): Promise<World> {
  if (!world.get('newInvokation')) {
    return world;
  } else {
    // Lastly, check invariants each hold
    await Promise.all(
      world.get('expectations').map(expectation => {
        // Check the expectation holds
        return expectation.checker(world);
      })
    );

    return world.set('expectations', []);
  }
}

export async function checkInvariants(world: World): Promise<World> {
  if (!world.get('newInvokation')) {
    return world;
  } else {
    // Lastly, check invariants each hold
    await Promise.all(
      world.get('invariants').map(invariant => {
        // Check the invariant still holds
        if (!invariant.held) {
          return invariant.checker(world);
        }
      })
    );

    // Remove holds
    return world.update('invariants', invariants => {
      return invariants.map(invariant => {
        invariant.held = false;

        return invariant;
      });
    });
  }
}

export function describeUser(world: World, address: string): string {
  // Look up by alias
  let alias = Object.entries(world.settings.aliases).find(([name, aliasAddr]) => aliasAddr === address);
  if (alias) {
    return `${alias[0]} (${address.slice(0,6)}...)`;
  }

  // Look up by `from`
  if (world.settings.from === address) {
    return `root (${address.slice(0,6)}...)`;
  }

  // Look up by unlocked accounts
  let account = world.accounts.find(account => account.address === address);
  if (account) {
    return `${account.name} (${address.slice(0,6)}...)`;
  }

  // Otherwise, just return the address itself
  return address;
}

// Fails an assertion with reason
export function fail(world: World, reason: string): World {
  if (world.event) {
    world.expect(undefined).fail(`${reason} processing ${formatEvent(world.event)}`);
  } else {
    world.expect(undefined).fail(reason);
  }

  return world;
}

import { Event } from "../Event";
import { addAction, World } from "../World";
import { GovernorBravo } from "../Contract/GovernorBravo";
import { buildGovernor } from "../Builder/GovernorBravoBuilder";
import { invoke } from "../Invokation";
import {
  getAddressV,
  getArrayV,
  getEventV,
  getNumberV,
  getStringV,
  getCoreValue,
} from "../CoreValue";
import { AddressV, ArrayV, EventV, NumberV, StringV } from "../Value";
import { Arg, Command, processCommandEvent, View } from "../Command";
import { verify } from "../Verify";
import { encodedNumber } from "../Encoding";
import { processProposalEvent } from "./BravoProposalEvent";
import { encodeParameters, rawValues } from "../Utils";
import { getGovernorV } from "../Value/GovernorBravoValue";
import { mergeContractABI } from "../Networks";

async function genGovernor(
  world: World,
  from: string,
  params: Event
): Promise<World> {
  let { world: nextWorld, governor, govData } = await buildGovernor(
    world,
    from,
    params
  );
  world = nextWorld;

  return addAction(
    world,
    `Deployed GovernorBravo ${govData.contract} to address ${governor._address}`,
    govData.invokation
  );
}

async function verifyGovernor(
  world: World,
  governor: GovernorBravo,
  apiKey: string,
  modelName: string,
  contractName: string
): Promise<World> {
  if (world.isLocalNetwork()) {
    world.printer.printLine(
      `Politely declining to verify on local network: ${world.network}.`
    );
  } else {
    await verify(world, apiKey, modelName, contractName, governor._address);
  }

  return world;
}

async function mergeABI(
  world: World,
  from: string,
  governorDelegator: GovernorBravo,
  governorDelegate: GovernorBravo
): Promise<World> {
  if (!world.dryRun) {
    // Skip this specifically on dry runs since it's likely to crash due to a number of reasons
    world = await mergeContractABI(
      world,
      "BravoDelegator",
      governorDelegator,
      governorDelegator.name,
      governorDelegate.name
    );
  }

  return world;
}

async function propose(
  world: World,
  from: string,
  governor: GovernorBravo,
  targets: string[],
  values: encodedNumber[],
  signatures: string[],
  calldatas: string[],
  description: string
): Promise<World> {
  const invokation = await invoke(
    world,
    governor.methods.propose(
      targets,
      values,
      signatures,
      calldatas,
      description
    ),
    from
  );
  return addAction(
    world,
    `Created new proposal "${description}" with id=${invokation.value} in Governor`,
    invokation
  );
}

async function setVotingDelay(
  world: World,
  from: string,
  governor: GovernorBravo,
  newVotingDelay: NumberV
): Promise<World> {
  let invokation = await invoke(
    world,
    governor.methods._setVotingDelay(newVotingDelay.encode()),
    from
  );

  world = addAction(
    world,
    `Set voting delay to ${newVotingDelay.show()}`,
    invokation
  );

  return world;
}

async function setVotingPeriod(
  world: World,
  from: string,
  governor: GovernorBravo,
  newVotingPeriod: NumberV
): Promise<World> {
  let invokation = await invoke(
    world,
    governor.methods._setVotingPeriod(newVotingPeriod.encode()),
    from
  );

  world = addAction(
    world,
    `Set voting period to ${newVotingPeriod.show()}`,
    invokation
  );

  return world;
}

async function setProposalThreshold(
  world: World,
  from: string,
  governor: GovernorBravo,
  newProposalThreshold: NumberV
): Promise<World> {
  let invokation = await invoke(
    world,
    governor.methods._setProposalThreshold(newProposalThreshold.encode()),
    from
  );

  world = addAction(
    world,
    `Set proposal threshold to ${newProposalThreshold.show()}`,
    invokation
  );

  return world;
}

async function setImplementation(
  world: World,
  from: string,
  governor: GovernorBravo,
  newImplementation: GovernorBravo
): Promise<World> {
  let invokation = await invoke(
    world,
    governor.methods._setImplementation(newImplementation._address),
    from
  );

  world = addAction(
    world,
    `Set GovernorBravo implementation to ${newImplementation}`,
    invokation
  );

  mergeABI(world, from, governor, newImplementation)

  return world;
}

async function initiate(
  world: World,
  from: string,
  governor: GovernorBravo,
  governorAlpha: string
): Promise<World> {
  let invokation = await invoke(
    world,
    governor.methods._initiate(governorAlpha),
    from
  );

  world = addAction(
    world,
    `Initiated governor from GovernorAlpha at ${governorAlpha}`,
    invokation
  );

  return world;
}

async function harnessInitiate(
  world: World,
  from: string,
  governor: GovernorBravo
): Promise<World> {
  let invokation = await invoke(world, governor.methods._initiate(), from);

  world = addAction(
    world,
    `Initiated governor using harness function`,
    invokation
  );

  return world;
}

async function setPendingAdmin(
  world: World,
  from: string,
  governor: GovernorBravo,
  newPendingAdmin: string
): Promise<World> {
  let invokation = await invoke(
    world,
    governor.methods._setPendingAdmin(newPendingAdmin),
    from
  );

  world = addAction(
    world,
    `Governor pending admin set to ${newPendingAdmin}`,
    invokation
  );

  return world;
}

async function acceptAdmin(
  world: World,
  from: string,
  governor: GovernorBravo
): Promise<World> {
  let invokation = await invoke(world, governor.methods._acceptAdmin(), from);

  world = addAction(world, `Governor admin accepted`, invokation);

  return world;
}

async function setBlockNumber(
  world: World,
  from: string,
  governor: GovernorBravo,
  blockNumber: NumberV
): Promise<World> {
  return addAction(
    world,
    `Set Governor blockNumber to ${blockNumber.show()}`,
    await invoke(
      world,
      governor.methods.setBlockNumber(blockNumber.encode()),
      from
    )
  );
}

async function setBlockTimestamp(
  world: World,
  from: string,
  governor: GovernorBravo,
  blockTimestamp: NumberV
): Promise<World> {
  return addAction(
    world,
    `Set Governor blockTimestamp to ${blockTimestamp.show()}`,
    await invoke(
      world,
      governor.methods.setBlockTimestamp(blockTimestamp.encode()),
      from
    )
  );
}

export function governorBravoCommands() {
  return [
    new Command<{ params: EventV }>(
      `
        #### Deploy

        * "Deploy ...params" - Generates a new Governor
        * E.g. "Governor Deploy BravoDelegate"
      `,
      "Deploy",
      [new Arg("params", getEventV, { variadic: true })],
      (world, from, { params }) => genGovernor(world, from, params.val)
    ),

    new Command<{
      governor: GovernorBravo;
      description: StringV;
      targets: ArrayV<AddressV>;
      values: ArrayV<NumberV>;
      signatures: ArrayV<StringV>;
      callDataArgs: ArrayV<ArrayV<StringV>>;
    }>(
      `
        #### Propose

        * "Governor <Governor> Propose description:<String> targets:<List> signatures:<List> callDataArgs:<List>" - Creates a new proposal in Governor
        * E.g. "Governor GovernorScenario Propose "New Interest Rate" [(Address cDAI)] [0] [("_setInterestRateModel(address)")] [[(Address MyInterestRateModel)]]
      `,
      "Propose",
      [
        new Arg("governor", getGovernorV),
        new Arg("description", getStringV),
        new Arg("targets", getArrayV(getAddressV)),
        new Arg("values", getArrayV(getNumberV)),
        new Arg("signatures", getArrayV(getStringV)),
        new Arg("callDataArgs", getArrayV(getArrayV(getCoreValue))),
      ],
      async (
        world,
        from,
        { governor, description, targets, values, signatures, callDataArgs }
      ) => {
        const targetsU = targets.val.map((a) => a.val);
        const valuesU = values.val.map((a) => a.encode());
        const signaturesU = signatures.val.map((a) => a.val);
        const callDatasU: string[] = signatures.val.reduce((acc, cur, idx) => {
          const args = rawValues(callDataArgs.val[idx]);
          acc.push(encodeParameters(world, cur.val, args));
          return acc;
        }, <string[]>[]);
        return await propose(
          world,
          from,
          governor,
          targetsU,
          valuesU,
          signaturesU,
          callDatasU,
          description.val
        );
      },
      { namePos: 1 }
    ),
    new Command<{ governor: GovernorBravo; params: EventV }>(
      `
        #### Proposal

        * "GovernorBravo <Governor> Proposal <...proposalEvent>" - Returns information about a proposal
        * E.g. "GovernorBravo GovernorScenario Proposal LastProposal Vote For"
      `,
      "Proposal",
      [
        new Arg("governor", getGovernorV),
        new Arg("params", getEventV, { variadic: true }),
      ],
      (world, from, { governor, params }) =>
        processProposalEvent(world, governor, params.val, from),
      { namePos: 1 }
    ),
    new Command<{ governor: GovernorBravo; blockNumber: NumberV }>(
      `
        #### SetBlockNumber

        * "Governor <Governor> SetBlockNumber <Seconds>" - Sets the blockNumber of the Governance Harness
        * E.g. "GovernorBravo SetBlockNumber 500"
    `,
      "SetBlockNumber",
      [new Arg("governor", getGovernorV), new Arg("blockNumber", getNumberV)],
      (world, from, { governor, blockNumber }) =>
        setBlockNumber(world, from, governor, blockNumber),
      { namePos: 1 }
    ),
    new Command<{ governor: GovernorBravo; blockTimestamp: NumberV }>(
      `
        #### SetBlockTimestamp

        * "Governor <Governor> SetBlockNumber <Seconds>" - Sets the blockTimestamp of the Governance Harness
        * E.g. "GovernorBravo GovernorScenario SetBlockTimestamp 500"
    `,
      "SetBlockTimestamp",
      [
        new Arg("governor", getGovernorV),
        new Arg("blockTimestamp", getNumberV),
      ],
      (world, from, { governor, blockTimestamp }) =>
        setBlockTimestamp(world, from, governor, blockTimestamp),
      { namePos: 1 }
    ),

    new Command<{ governor: GovernorBravo; newVotingDelay: NumberV }>(
      `
        #### SetVotingDelay

        * "GovernorBravo <Governor> SetVotingDelay <Blocks>" - Sets the voting delay of the GovernorBravo
        * E.g. "GovernorBravo GovernorBravoScenario SetVotingDelay 2"
    `,
      "SetVotingDelay",
      [
        new Arg("governor", getGovernorV),
        new Arg("newVotingDelay", getNumberV),
      ],
      (world, from, { governor, newVotingDelay }) =>
        setVotingDelay(world, from, governor, newVotingDelay),
      { namePos: 1 }
    ),
    new Command<{ governor: GovernorBravo; newVotingPeriod: NumberV }>(
      `
        #### SetVotingPeriod

        * "GovernorBravo <Governor> SetVotingPeriod <Blocks>" - Sets the voting period of the GovernorBravo
        * E.g. "GovernorBravo GovernorBravoScenario SetVotingPeriod 500"
    `,
      "SetVotingPeriod",
      [
        new Arg("governor", getGovernorV),
        new Arg("newVotingPeriod", getNumberV),
      ],
      (world, from, { governor, newVotingPeriod }) =>
        setVotingPeriod(world, from, governor, newVotingPeriod),
      { namePos: 1 }
    ),
    new Command<{ governor: GovernorBravo; newProposalThreshold: NumberV }>(
      `
        #### SetProposalThreshold

        * "GovernorBravo <Governor> SetProposalThreshold <Comp>" - Sets the proposal threshold of the GovernorBravo
        * E.g. "GovernorBravo GovernorBravoScenario SetProposalThreshold 500e18"
    `,
      "SetProposalThreshold",
      [
        new Arg("governor", getGovernorV),
        new Arg("newProposalThreshold", getNumberV),
      ],
      (world, from, { governor, newProposalThreshold }) =>
        setProposalThreshold(world, from, governor, newProposalThreshold),
      { namePos: 1 }
    ),
    new Command<{ governor: GovernorBravo; governorAlpha: AddressV }>(
      `
        #### Initiate

        * "GovernorBravo <Governor> Initiate <AddressV>" - Initiates the Governor relieving the given GovernorAlpha
        * E.g. "GovernorBravo GovernorBravoScenario Initiate (Address GovernorAlpha)"
    `,
      "Initiate",
      [
        new Arg("governor", getGovernorV),
        new Arg("governorAlpha", getAddressV),
      ],
      (world, from, { governor, governorAlpha }) =>
        initiate(world, from, governor, governorAlpha.val),
      { namePos: 1 }
    ),
    new Command<{ governor: GovernorBravo }>(
      `
        #### HarnessInitiate

        * "GovernorBravo <Governor> HarnessInitiate" - Uses harness function to bypass initiate for testing
        * E.g. "GovernorBravo GovernorBravoScenario HarnessInitiate"
    `,
      "HarnessInitiate",
      [new Arg("governor", getGovernorV)],
      (world, from, { governor }) => harnessInitiate(world, from, governor),
      { namePos: 1 }
    ),
    new Command<{ governor: GovernorBravo; newImplementation: GovernorBravo }>(
      `
        #### SetImplementation

        * "GovernorBravo <Governor> SetImplementation <Governor>" - Sets the address for the GovernorBravo implementation
        * E.g. "GovernorBravo GovernorBravoScenario SetImplementation newImplementation"
    `,
      "SetImplementation",
      [
        new Arg("governor", getGovernorV),
        new Arg("newImplementation", getGovernorV),
      ],
      (world, from, { governor, newImplementation }) =>
        setImplementation(world, from, governor, newImplementation),
      { namePos: 1 }
    ),
    new Command<{ governor: GovernorBravo; newPendingAdmin: AddressV }>(
      `
        #### SetPendingAdmin

        * "GovernorBravo <Governor> SetPendingAdmin <AddressV>" - Sets the address for the GovernorBravo pending admin
        * E.g. "GovernorBravo GovernorBravoScenario SetPendingAdmin newAdmin"
    `,
      "SetPendingAdmin",
      [
        new Arg("governor", getGovernorV),
        new Arg("newPendingAdmin", getAddressV),
      ],
      (world, from, { governor, newPendingAdmin }) =>
        setPendingAdmin(world, from, governor, newPendingAdmin.val),
      { namePos: 1 }
    ),
    new Command<{ governor: GovernorBravo }>(
      `
        #### AcceptAdmin

        * "GovernorBravo <Governor> AcceptAdmin" - Pending admin accepts the admin role
        * E.g. "GovernorBravo GovernorBravoScenario AcceptAdmin"
    `,
      "AcceptAdmin",
      [new Arg("governor", getGovernorV)],
      (world, from, { governor }) => acceptAdmin(world, from, governor),
      { namePos: 1 }
    ),

    new Command<{
      governorDelegator: GovernorBravo;
      governorDelegate: GovernorBravo;
    }>(
      `#### MergeABI

        * "ComptrollerImpl <Impl> MergeABI" - Merges the ABI, as if it was a become.
        * E.g. "ComptrollerImpl MyImpl MergeABI
      `,
      "MergeABI",
      [
        new Arg("governorDelegator", getGovernorV),
        new Arg("governorDelegate", getGovernorV),
      ],
      (world, from, { governorDelegator, governorDelegate }) =>
        mergeABI(world, from, governorDelegator, governorDelegate),
      { namePos: 1 }
    ),
  ];
}

export async function processGovernorBravoEvent(
  world: World,
  event: Event,
  from: string | null
): Promise<World> {
  return await processCommandEvent<any>(
    "Governor",
    governorBravoCommands(),
    world,
    event,
    from
  );
}

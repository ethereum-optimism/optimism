import { Event } from '../Event';
import { addAction, describeUser, World } from '../World';
import { Governor } from '../Contract/Governor';
import { invoke } from '../Invokation';
import {
  getEventV,
} from '../CoreValue';
import {
  EventV,
} from '../Value';
import { Arg, Command, processCommandEvent } from '../Command';
import { getProposalId } from '../Value/ProposalValue';

function getSupport(support: Event): boolean {
  if (typeof support === 'string') {
    if (support === 'For' || support === 'Against') {
      return support === 'For';
    }
  }

  throw new Error(`Unknown support flag \`${support}\`, expected "For" or "Against"`);
}

async function describeProposal(world: World, governor: Governor, proposalId: number): Promise<string> {
  // const proposal = await governor.methods.proposals(proposalId).call();
  return `proposal ${proposalId.toString()}`; // TODO: Cleanup
}

export function proposalCommands(governor: Governor) {
  return [
    new Command<{ proposalIdent: EventV, support: EventV }>(
      `
        #### Vote

        * "Governor <Governor> Vote <For|Against>" - Votes for or against a given proposal
        * E.g. "Governor GovernorScenario Proposal LastProposal Vote For"
    `,
      'Vote',
      [
        new Arg("proposalIdent", getEventV),
        new Arg("support", getEventV),
      ],
      async (world, from, { proposalIdent, support }) => {
        const proposalId = await getProposalId(world, governor, proposalIdent.val);
        const invokation = await invoke(world, governor.methods.castVote(proposalId, getSupport(support.val)), from);

        return addAction(
          world,
          `Cast ${support.val.toString()} vote from ${describeUser(world, from)} for proposal ${proposalId}`,
          invokation
        )
      },
      { namePos: 1 }
    ),

    new Command<{ proposalIdent: EventV }>(
      `
        #### Queue
        * "Governor <Governor> Queue" - Queues given proposal
        * E.g. "Governor GovernorScenario Proposal LastProposal Queue"
    `,
      'Queue',
      [
        new Arg("proposalIdent", getEventV)
      ],
      async (world, from, { proposalIdent }) => {
        const proposalId = await getProposalId(world, governor, proposalIdent.val);
        const invokation = await invoke(world, governor.methods.queue(proposalId), from);

        return addAction(
          world,
          `Queue proposal ${await describeProposal(world, governor, proposalId)} from ${describeUser(world, from)}`,
          invokation
        )
      },
      { namePos: 1 }
    ),
    new Command<{ proposalIdent: EventV }>(
      `
        #### Execute
        * "Governor <Governor> Execute" - Executes given proposal
        * E.g. "Governor GovernorScenario Proposal LastProposal Execute"
    `,
      'Execute',
      [
        new Arg("proposalIdent", getEventV)
      ],
      async (world, from, { proposalIdent }) => {
        const proposalId = await getProposalId(world, governor, proposalIdent.val);
        const invokation = await invoke(world, governor.methods.execute(proposalId), from);

        return addAction(
          world,
          `Execute proposal ${await describeProposal(world, governor, proposalId)} from ${describeUser(world, from)}`,
          invokation
        )
      },
      { namePos: 1 }
    ),
    new Command<{ proposalIdent: EventV }>(
      `
        #### Cancel
        * "Cancel" - cancels given proposal
        * E.g. "Governor Proposal LastProposal Cancel"
    `,
      'Cancel',
      [
        new Arg("proposalIdent", getEventV)
      ],
      async (world, from, { proposalIdent }) => {
        const proposalId = await getProposalId(world, governor, proposalIdent.val);
        const invokation = await invoke(world, governor.methods.cancel(proposalId), from);

        return addAction(
          world,
          `Cancel proposal ${await describeProposal(world, governor, proposalId)} from ${describeUser(world, from)}`,
          invokation
        )
      },
      { namePos: 1 }
    ),
  ];
}

export async function processProposalEvent(world: World, governor: Governor, event: Event, from: string | null): Promise<World> {
  return await processCommandEvent<any>('Proposal', proposalCommands(governor), world, event, from);
}

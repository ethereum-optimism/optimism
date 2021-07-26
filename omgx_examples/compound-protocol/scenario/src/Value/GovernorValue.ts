import { Event } from '../Event';
import { World } from '../World';
import { Governor } from '../Contract/Governor';
import {
  getCoreValue,
  getEventV,
  mapValue
} from '../CoreValue';
import {
  AddressV,
  EventV,
  Value
} from '../Value';
import { Arg, Fetcher, getFetcherValue } from '../Command';
import { getProposalValue } from './ProposalValue';
import { getGovernorAddress, getWorldContractByAddress } from '../ContractLookup';

export async function getGovernorV(world: World, event: Event): Promise<Governor> {
  const address = await mapValue<AddressV>(
    world,
    event,
    (str) => new AddressV(getGovernorAddress(world, str)),
    getCoreValue,
    AddressV
  );

  return getWorldContractByAddress<Governor>(world, address.val);
}

export async function governorAddress(world: World, governor: Governor): Promise<AddressV> {
  return new AddressV(governor._address);
}

export async function getGovernorGuardian(world: World, governor: Governor): Promise<AddressV> {
  return new AddressV(await governor.methods.guardian().call());
}

export function governorFetchers() {
  return [
    new Fetcher<{ governor: Governor }, AddressV>(`
        #### Address

        * "Governor <Governor> Address" - Returns the address of governor contract
          * E.g. "Governor GovernorScenario Address"
      `,
      "Address",
      [
        new Arg("governor", getGovernorV)
      ],
      (world, { governor }) => governorAddress(world, governor),
      { namePos: 1 }
    ),

    new Fetcher<{ governor: Governor }, AddressV>(`
        #### Guardian

        * "Governor <Governor> Guardian" - Returns the address of governor guardian
          * E.g. "Governor GovernorScenario Guardian"
      `,
      "Guardian",
      [
        new Arg("governor", getGovernorV)
      ],
      (world, { governor }) => getGovernorGuardian(world, governor),
      { namePos: 1 }
    ),

    new Fetcher<{ governor: Governor, params: EventV }, Value>(`
        #### Proposal

        * "Governor <Governor> Proposal <...proposalValue>" - Returns information about a proposal
          * E.g. "Governor GovernorScenario Proposal LastProposal Id"
      `,
      "Proposal",
      [
        new Arg("governor", getGovernorV),
        new Arg("params", getEventV, { variadic: true })
      ],
      (world, { governor, params }) => getProposalValue(world, governor, params.val),
      { namePos: 1 }
    ),
  ];
}

export async function getGovernorValue(world: World, event: Event): Promise<Value> {
  return await getFetcherValue<any, any>("Governor", governorFetchers(), world, event);
}

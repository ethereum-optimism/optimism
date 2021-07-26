import { Event } from '../Event';
import { World } from '../World';
import { Comp } from '../Contract/Comp';
import {
  getAddressV,
  getNumberV
} from '../CoreValue';
import {
  AddressV,
  ListV,
  NumberV,
  StringV,
  Value
} from '../Value';
import { Arg, Fetcher, getFetcherValue } from '../Command';
import { getComp } from '../ContractLookup';

export function compFetchers() {
  return [
    new Fetcher<{ comp: Comp }, AddressV>(`
        #### Address

        * "<Comp> Address" - Returns the address of Comp token
          * E.g. "Comp Address"
      `,
      "Address",
      [
        new Arg("comp", getComp, { implicit: true })
      ],
      async (world, { comp }) => new AddressV(comp._address)
    ),

    new Fetcher<{ comp: Comp }, StringV>(`
        #### Name

        * "<Comp> Name" - Returns the name of the Comp token
          * E.g. "Comp Name"
      `,
      "Name",
      [
        new Arg("comp", getComp, { implicit: true })
      ],
      async (world, { comp }) => new StringV(await comp.methods.name().call())
    ),

    new Fetcher<{ comp: Comp }, StringV>(`
        #### Symbol

        * "<Comp> Symbol" - Returns the symbol of the Comp token
          * E.g. "Comp Symbol"
      `,
      "Symbol",
      [
        new Arg("comp", getComp, { implicit: true })
      ],
      async (world, { comp }) => new StringV(await comp.methods.symbol().call())
    ),

    new Fetcher<{ comp: Comp }, NumberV>(`
        #### Decimals

        * "<Comp> Decimals" - Returns the number of decimals of the Comp token
          * E.g. "Comp Decimals"
      `,
      "Decimals",
      [
        new Arg("comp", getComp, { implicit: true })
      ],
      async (world, { comp }) => new NumberV(await comp.methods.decimals().call())
    ),

    new Fetcher<{ comp: Comp }, NumberV>(`
        #### TotalSupply

        * "Comp TotalSupply" - Returns Comp token's total supply
      `,
      "TotalSupply",
      [
        new Arg("comp", getComp, { implicit: true })
      ],
      async (world, { comp }) => new NumberV(await comp.methods.totalSupply().call())
    ),

    new Fetcher<{ comp: Comp, address: AddressV }, NumberV>(`
        #### TokenBalance

        * "Comp TokenBalance <Address>" - Returns the Comp token balance of a given address
          * E.g. "Comp TokenBalance Geoff" - Returns Geoff's Comp balance
      `,
      "TokenBalance",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("address", getAddressV)
      ],
      async (world, { comp, address }) => new NumberV(await comp.methods.balanceOf(address.val).call())
    ),

    new Fetcher<{ comp: Comp, owner: AddressV, spender: AddressV }, NumberV>(`
        #### Allowance

        * "Comp Allowance owner:<Address> spender:<Address>" - Returns the Comp allowance from owner to spender
          * E.g. "Comp Allowance Geoff Torrey" - Returns the Comp allowance of Geoff to Torrey
      `,
      "Allowance",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("owner", getAddressV),
        new Arg("spender", getAddressV)
      ],
      async (world, { comp, owner, spender }) => new NumberV(await comp.methods.allowance(owner.val, spender.val).call())
    ),

    new Fetcher<{ comp: Comp, account: AddressV }, NumberV>(`
        #### GetCurrentVotes

        * "Comp GetCurrentVotes account:<Address>" - Returns the current Comp votes balance for an account
          * E.g. "Comp GetCurrentVotes Geoff" - Returns the current Comp vote balance of Geoff
      `,
      "GetCurrentVotes",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("account", getAddressV),
      ],
      async (world, { comp, account }) => new NumberV(await comp.methods.getCurrentVotes(account.val).call())
    ),

    new Fetcher<{ comp: Comp, account: AddressV, blockNumber: NumberV }, NumberV>(`
        #### GetPriorVotes

        * "Comp GetPriorVotes account:<Address> blockBumber:<Number>" - Returns the current Comp votes balance at given block
          * E.g. "Comp GetPriorVotes Geoff 5" - Returns the Comp vote balance for Geoff at block 5
      `,
      "GetPriorVotes",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("account", getAddressV),
        new Arg("blockNumber", getNumberV),
      ],
      async (world, { comp, account, blockNumber }) => new NumberV(await comp.methods.getPriorVotes(account.val, blockNumber.encode()).call())
    ),

    new Fetcher<{ comp: Comp, account: AddressV }, NumberV>(`
        #### GetCurrentVotesBlock

        * "Comp GetCurrentVotesBlock account:<Address>" - Returns the current Comp votes checkpoint block for an account
          * E.g. "Comp GetCurrentVotesBlock Geoff" - Returns the current Comp votes checkpoint block for Geoff
      `,
      "GetCurrentVotesBlock",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("account", getAddressV),
      ],
      async (world, { comp, account }) => {
        const numCheckpoints = Number(await comp.methods.numCheckpoints(account.val).call());
        const checkpoint = await comp.methods.checkpoints(account.val, numCheckpoints - 1).call();

        return new NumberV(checkpoint.fromBlock);
      }
    ),

    new Fetcher<{ comp: Comp, account: AddressV }, NumberV>(`
        #### VotesLength

        * "Comp VotesLength account:<Address>" - Returns the Comp vote checkpoint array length
          * E.g. "Comp VotesLength Geoff" - Returns the Comp vote checkpoint array length of Geoff
      `,
      "VotesLength",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("account", getAddressV),
      ],
      async (world, { comp, account }) => new NumberV(await comp.methods.numCheckpoints(account.val).call())
    ),

    new Fetcher<{ comp: Comp, account: AddressV }, ListV>(`
        #### AllVotes

        * "Comp AllVotes account:<Address>" - Returns information about all votes an account has had
          * E.g. "Comp AllVotes Geoff" - Returns the Comp vote checkpoint array
      `,
      "AllVotes",
      [
        new Arg("comp", getComp, { implicit: true }),
        new Arg("account", getAddressV),
      ],
      async (world, { comp, account }) => {
        const numCheckpoints = Number(await comp.methods.numCheckpoints(account.val).call());
        const checkpoints = await Promise.all(new Array(numCheckpoints).fill(undefined).map(async (_, i) => {
          const {fromBlock, votes} = await comp.methods.checkpoints(account.val, i).call();

          return new StringV(`Block ${fromBlock}: ${votes} vote${votes !== 1 ? "s" : ""}`);
        }));

        return new ListV(checkpoints);
      }
    )
  ];
}

export async function getCompValue(world: World, event: Event): Promise<Value> {
  return await getFetcherValue<any, any>("Comp", compFetchers(), world, event);
}

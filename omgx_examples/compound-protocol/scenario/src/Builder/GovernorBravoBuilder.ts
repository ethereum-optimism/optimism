import { Event } from "../Event";
import { World } from "../World";
import { GovernorBravo } from "../Contract/GovernorBravo";
import { Invokation } from "../Invokation";
import { getAddressV, getNumberV, getStringV } from "../CoreValue";
import { AddressV, NumberV, StringV } from "../Value";
import { Arg, Fetcher, getFetcherValue } from "../Command";
import { storeAndSaveContract } from "../Networks";
import { getContract } from "../Contract";

const GovernorBravoDelegate = getContract("GovernorBravoDelegate");
const GovernorBravoDelegateHarness = getContract("GovernorBravoDelegateHarness");
const GovernorBravoDelegator = getContract("GovernorBravoDelegator");
const GovernorBravoImmutable = getContract("GovernorBravoImmutable");

export interface GovernorBravoData {
  invokation: Invokation<GovernorBravo>;
  name: string;
  contract: string;
  address?: string;
}

export async function buildGovernor(
  world: World,
  from: string,
  params: Event
): Promise<{ world: World; governor: GovernorBravo; govData: GovernorBravoData }> {
  const fetchers = [
    new Fetcher<
      { name: StringV, timelock: AddressV, comp: AddressV, admin: AddressV, implementation: AddressV, votingPeriod: NumberV, votingDelay: NumberV, proposalThreshold: NumberV},
      GovernorBravoData
    >(
      `
      #### GovernorBravoDelegator

      * "GovernorBravo Deploy BravoDelegator name:<String> timelock:<Address> comp:<Address> admin:<Address> implementation<address> votingPeriod:<Number> votingDelay:<Number>" - Deploys Compound Governor Bravo with a given parameters
        * E.g. "GovernorBravo Deploy BravoDelegator GovernorBravo (Address Timelock) (Address Comp) Admin (Address impl) 17280 1"
    `,
      "BravoDelegator",
      [
        new Arg("name", getStringV),
        new Arg("timelock", getAddressV),
        new Arg("comp", getAddressV),
        new Arg("admin", getAddressV),
        new Arg("implementation", getAddressV),
        new Arg("votingPeriod", getNumberV),
        new Arg("votingDelay", getNumberV),
        new Arg("proposalThreshold", getNumberV)
      ],
      async (world, { name, timelock, comp, admin, implementation, votingPeriod, votingDelay, proposalThreshold }) => {
        return {
          invokation: await GovernorBravoDelegator.deploy<GovernorBravo>(
            world,
            from,
            [timelock.val, comp.val, admin.val, implementation.val, votingPeriod.encode(), votingDelay.encode(), proposalThreshold.encode()]
          ),
          name: name.val,
          contract: "GovernorBravoDelegator"
        };
      }
    ),
    new Fetcher<
      { name: StringV, timelock: AddressV, comp: AddressV, admin: AddressV, votingPeriod: NumberV, votingDelay: NumberV, proposalThreshold: NumberV },
      GovernorBravoData
    >(
      `
      #### GovernorBravoImmutable

      * "GovernorBravoImmut Deploy BravoImmutable name:<String> timelock:<Address> comp:<Address> admin:<Address> votingPeriod:<Number> votingDelay:<Number>" - Deploys Compound Governor Bravo Immutable with a given parameters
        * E.g. "GovernorBravo Deploy BravoImmutable GovernorBravo (Address Timelock) (Address Comp) Admin 17280 1"
    `,
      "BravoImmutable",
      [
        new Arg("name", getStringV),
        new Arg("timelock", getAddressV),
        new Arg("comp", getAddressV),
        new Arg("admin", getAddressV),
        new Arg("votingPeriod", getNumberV),
        new Arg("votingDelay", getNumberV),
        new Arg("proposalThreshold", getNumberV)
      ],
      async (world, { name, timelock, comp, admin, votingPeriod, votingDelay, proposalThreshold }) => {
        return {
          invokation: await GovernorBravoImmutable.deploy<GovernorBravo>(
            world,
            from,
            [timelock.val, comp.val, admin.val, votingPeriod.encode(), votingDelay.encode(), proposalThreshold.encode()]
          ),
          name: name.val,
          contract: "GovernorBravoImmutable"
        };
      }
    ),
    new Fetcher<
      { name: StringV },
      GovernorBravoData
    >(
      `
      #### GovernorBravoDelegate

      * "Governor Deploy BravoDelegate name:<String>" - Deploys Compound Governor Bravo Delegate
        * E.g. "Governor Deploy BravoDelegate GovernorBravoDelegate"
    `,
      "BravoDelegate",
      [
        new Arg("name", getStringV)
      ],
      async (world, { name }) => {
        return {
          invokation: await GovernorBravoDelegate.deploy<GovernorBravo>(
            world,
            from,
            []
          ),
          name: name.val,
          contract: "GovernorBravoDelegate"
        };
      }
    ),
    new Fetcher<
      { name: StringV },
      GovernorBravoData
    >(
      `
      #### GovernorBravoDelegateHarness

      * "Governor Deploy BravoDelegateHarness name:<String>" - Deploys Compound Governor Bravo Delegate Harness
        * E.g. "Governor Deploy BravoDelegateHarness GovernorBravoDelegateHarness"
    `,
      "BravoDelegateHarness",
      [
        new Arg("name", getStringV)
      ],
      async (world, { name }) => {
        return {
          invokation: await GovernorBravoDelegateHarness.deploy<GovernorBravo>(
            world,
            from,
            []
          ),
          name: name.val,
          contract: "GovernorBravoDelegateHarness"
        };
      }
    )
  ];

  let govData = await getFetcherValue<any, GovernorBravoData>(
    "DeployGovernor",
    fetchers,
    world,
    params
  );
  let invokation = govData.invokation;
  delete govData.invokation;

  if (invokation.error) {
    throw invokation.error;
  }

  const governor = invokation.value!;
  govData.address = governor._address;

  world = await storeAndSaveContract(
    world,
    governor,
    govData.name,
    invokation,
    [
      { index: ["Governor", govData.name], data: govData },
    ]
  );

  return { world, governor, govData };
}

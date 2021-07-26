import {Event} from '../Event';
import {addAction, World} from '../World';
import {InterestRateModel} from '../Contract/InterestRateModel';
import {Invokation, invoke} from '../Invokation';
import {
  getAddressV,
  getExpNumberV,
  getNumberV,
  getPercentV,
  getStringV,
} from '../CoreValue';
import {
  AddressV,
  EventV,
  NumberV,
  StringV,
} from '../Value';
import {Arg, Fetcher, getFetcherValue} from '../Command';
import {storeAndSaveContract} from '../Networks';
import {getContract, getTestContract} from '../Contract';

const FixedInterestRateModel = getTestContract('InterestRateModelHarness');
const WhitePaperInterestRateModel = getContract('WhitePaperInterestRateModel');
const JumpRateModel = getContract('JumpRateModel');
const DAIInterestRateModel = getContract('DAIInterestRateModelV3');
const JumpRateModelV2 = getContract('JumpRateModelV2');
const LegacyJumpRateModelV2 = getContract('LegacyJumpRateModelV2');

export interface InterestRateModelData {
  invokation: Invokation<InterestRateModel>
  address?: string
  name: string
  contract: string
  description: string
  base?: string
  slope?: string
  kink?: string
  jump?: string
}

export async function buildInterestRateModel(world: World, from: string, event: Event): Promise<{world: World, interestRateModel: InterestRateModel, interestRateModelData: InterestRateModelData}> {
  const fetchers = [
    new Fetcher<{name: StringV, rate: NumberV}, InterestRateModelData>(`
        #### Fixed

        * "Fixed name:<String> rate:<Number>" - Fixed interest **per-block** rate
          * E.g. "InterestRateModel Deploy Fixed MyInterestRateModel 0.5"
      `,
      "Fixed",
      [
        new Arg("name", getStringV),
        new Arg("rate", getPercentV),
      ],
      async (world, {name, rate}) => ({
        invokation: await FixedInterestRateModel.deploy<InterestRateModel>(world, from, [rate.encode()]),
        name: name.val,
        contract: "InterestRateModelHarness",
        description: `Fixed rate ${rate.show()} per block`
      })
    ),

    new Fetcher<{name: StringV, baseRate: NumberV, multiplier: NumberV}, InterestRateModelData>(`
        #### WhitePaper

        * "WhitePaper name:<String> baseRate:<Number> multiplier:<Number>" - The WhitePaper interest rate
          * E.g. "InterestRateModel Deploy WhitePaper MyInterestRateModel 0.05 0.2" - 5% base rate and 20% utilization multiplier
      `,
      "WhitePaper",
      [
        new Arg("name", getStringV),
        new Arg("baseRate", getExpNumberV),
        new Arg("multiplier", getExpNumberV)
      ],
      async (world, {name, baseRate, multiplier}) => ({
        invokation: await WhitePaperInterestRateModel.deploy<InterestRateModel>(world, from, [baseRate.encode(), multiplier.encode()]),
        name: name.val,
        contract: "WhitePaperInterestRateModel",
        description: `WhitePaper baseRate=${baseRate.encode().toString()} multiplier=${multiplier.encode().toString()}`,
        base: baseRate.encode().toString(),
        slope: multiplier.encode().toString()
      })
    ),

    new Fetcher<{name: StringV, baseRate: NumberV, multiplier: NumberV, jump: NumberV, kink: NumberV}, InterestRateModelData>(`
         #### JumpRateModel

         * "JumpRateModel name:<String> baseRate:<Number> multiplier:<Number> jump:<Number> kink:<Number>" - The Jump interest rate
           * E.g. "InterestRateModel Deploy JumpRateModel MyInterestRateModel 0.05 0.2 2 0.90" - 5% base rate and 20% utilization multiplier and 200% multiplier at 90% utilization
       `,
       "JumpRateModel",
       [
         new Arg("name", getStringV),
         new Arg("baseRate", getExpNumberV),
         new Arg("multiplier", getExpNumberV),
         new Arg("jump", getExpNumberV),
         new Arg("kink", getExpNumberV)
       ],
       async (world, {name, baseRate, multiplier, jump, kink}) => ({
         invokation: await JumpRateModel.deploy<InterestRateModel>(world, from, [baseRate.encode(), multiplier.encode(), jump.encode(), kink.val]),
         name: name.val,
         contract: "JumpRateModel",
         description: `JumpRateModel baseRate=${baseRate.encode().toString()} multiplier=${multiplier.encode().toString()} jump=${jump.encode().toString()} kink=${kink.encode().toString()}`,
         base: baseRate.encode().toString(),
         slope: multiplier.encode().toString(),
         jump: jump.encode().toString(),
         kink: kink.encode().toString()
       })
    ),

    new Fetcher<{name: StringV, baseRate: NumberV, multiplier: NumberV, jump: NumberV, kink: NumberV, owner: AddressV}, InterestRateModelData>(`
         #### JumpRateModelV2

         * "JumpRateModelV2 name:<String> baseRate:<Number> multiplier:<Number> jump:<Number> kink:<Number> owner:<Address>" - The v2 Jump interest rate model
           * E.g. "InterestRateModel Deploy JumpRateModelV2 MyInterestRateModel 0.05 0.2 2 0.90 (Address Timelock)" - 5% base rate and 20% + 5% interest at kink and 200% multiplier starting at the kink of 90% utilization
       `,
       "JumpRateModelV2",
       [
         new Arg("name", getStringV),
         new Arg("baseRate", getExpNumberV),
         new Arg("multiplier", getExpNumberV),
         new Arg("jump", getExpNumberV),
         new Arg("kink", getExpNumberV),
         new Arg("owner", getAddressV),
       ],
       async (world, {name, baseRate, multiplier, jump, kink, owner}) => ({
         invokation: await JumpRateModelV2.deploy<InterestRateModel>(world, from, [baseRate.encode(), multiplier.encode(), jump.encode(), kink.encode(), owner.val]),
         name: name.val,
         contract: "JumpRateModelV2",
         description: `JumpRateModelV2 baseRate=${baseRate.encode().toString()} multiplier=${multiplier.encode().toString()} jump=${jump.encode().toString()} kink=${kink.encode().toString()}`,
         base: baseRate.encode().toString(),
         slope: multiplier.encode().toString(),
         jump: jump.encode().toString(),
         kink: kink.encode().toString(),
         owner: owner.val,
       })
    ),

    new Fetcher<{name: StringV, baseRate: NumberV, multiplier: NumberV, jump: NumberV, kink: NumberV, owner: AddressV}, InterestRateModelData>(`
         #### LegacyJumpRateModelV2

         * "LegacyJumpRateModelV2 name:<String> baseRate:<Number> multiplier:<Number> jump:<Number> kink:<Number> owner:<Address>" - The legacy v2 Jump interest rate model
           * E.g. "InterestRateModel Deploy LegacyJumpRateModelV2 MyInterestRateModel 0.05 0.2 2 0.90 (Address Timelock)" - 5% base rate and 20% + 5% interest at kink and 200% multiplier starting at the kink of 90% utilization
       `,
       "LegacyJumpRateModelV2",
       [
         new Arg("name", getStringV),
         new Arg("baseRate", getExpNumberV),
         new Arg("multiplier", getExpNumberV),
         new Arg("jump", getExpNumberV),
         new Arg("kink", getExpNumberV),
         new Arg("owner", getAddressV),
       ],
       async (world, {name, baseRate, multiplier, jump, kink, owner}) => ({
         invokation: await LegacyJumpRateModelV2.deploy<InterestRateModel>(world, from, [baseRate.encode(), multiplier.encode(), jump.encode(), kink.encode(), owner.val]),
         name: name.val,
         contract: "LegacyJumpRateModelV2",
         description: `LegacyJumpRateModelV2 baseRate=${baseRate.encode().toString()} multiplier=${multiplier.encode().toString()} jump=${jump.encode().toString()} kink=${kink.encode().toString()}`,
         base: baseRate.encode().toString(),
         slope: multiplier.encode().toString(),
         jump: jump.encode().toString(),
         kink: kink.encode().toString(),
         owner: owner.val,
       })
    ),

    new Fetcher<{name: StringV, jump: NumberV, kink: NumberV, pot: AddressV, jug: AddressV, owner: AddressV}, InterestRateModelData>(`
         #### DAIInterestRateModel

         * "DAIInterestRateModel name:<String> jump:<Number> kink:<Number> pot:<Address> jug:<Address> owner:<Address>" - The DAI interest rate model
           * E.g. "InterestRateModel Deploy DAIInterestRateModel MyInterestRateModel (Exp 2) (Exp 0.9) PotAddress JugAddress" Timelock - 200% multiplier at 90% utilization
       `,
       "DAIInterestRateModel",
       [
         new Arg("name", getStringV),
         new Arg("jump", getExpNumberV),
         new Arg("kink", getExpNumberV),
         new Arg("pot", getAddressV),
         new Arg("jug", getAddressV),
         new Arg("owner", getAddressV),
       ],
       async (world, {name, jump, kink, pot, jug, owner}) => ({
         invokation: await DAIInterestRateModel.deploy<InterestRateModel>(world, from, [jump.encode(), kink.encode(), pot.val, jug.val, owner.val]),
         name: name.val,
         contract: "DAIInterestRateModel",
         description: `DAIInterestRateModel jump=${jump.encode().toString()} kink=${kink.encode().toString()} pot=${pot.val} jug=${jug.val} owner=${owner.val}`,
         jump: jump.encode().toString(),
         kink: kink.encode().toString(),
         pot: pot.val,
         jug: jug.val,
         owner: owner.val
       })
     )
  ];

  let interestRateModelData = await getFetcherValue<any, InterestRateModelData>("DeployInterestRateModel", fetchers, world, event);
  let invokation = interestRateModelData.invokation;
  delete interestRateModelData.invokation;

  if (invokation.error) {
    throw invokation.error;
  }
  const interestRateModel = invokation.value!;
  interestRateModelData.address = interestRateModel._address;

  world = await storeAndSaveContract(
    world,
    interestRateModel,
    interestRateModelData.name,
    invokation,
    [
      {
        index: ['InterestRateModel', interestRateModelData.name],
        data: interestRateModelData
      }
    ]
  );

  return {world, interestRateModel, interestRateModelData};
}

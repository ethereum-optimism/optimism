import {Event} from '../Event';
import {addExpectation, World} from '../World';
import {
  EventV,
  NumberV,
  Value
} from '../Value';
import {
  getCoreValue,
  getEventV,
  getNumberV
} from '../CoreValue';
import {Invariant} from '../Invariant';
import {ChangesExpectation} from '../Expectation/ChangesExpectation';
import {RemainsExpectation} from '../Expectation/RemainsExpectation';
import {formatEvent} from '../Formatter';
import {Arg, View, processCommandEvent} from '../Command';

async function changesExpectation(world: World, condition: Event, delta: NumberV, tolerance: NumberV): Promise<World> {
  const value = await getCoreValue(world, condition);
  const expectation = new ChangesExpectation(condition, value, delta, tolerance);

  return addExpectation(world, expectation);
}

async function remainsExpectation(world: World, condition: Event, value: Value): Promise<World> {
  const expectation = new RemainsExpectation(condition, value);

  // Immediately check value matches
  await expectation.checker(world, true);

  return addExpectation(world, expectation);
}

export function expectationCommands() {
  return [
    new View<{condition: EventV, delta: NumberV, tolerance: NumberV}>(`
        #### Changes

        * "Changes <Value> amount:<Number> tolerance:<Number>" - Expects that given value changes by amount
          * E.g ."Expect Changes (CToken cZRX UnderlyingBalance Geoff) +10e18"
      `,
      "Changes",
      [
        new Arg("condition", getEventV),
        new Arg("delta", getNumberV),
        new Arg("tolerance", getNumberV, {default: new NumberV(0)})
      ],
      (world, {condition, delta, tolerance}) => changesExpectation(world, condition.val, delta, tolerance)
    ),

    new View<{condition: EventV, value: Value}>(`
        #### Remains

        * "Expect Remains <Condition> <Value>" - Ensures that the given condition starts at and remains a given value
          * E.g ."Expect Remains (CToken cZRX UnderlyingBalance Geoff) (Exactly 0)"
      `,
      "Remains",
      [
        new Arg("condition", getEventV),
        new Arg("value", getCoreValue)
      ],
      (world, {condition, value}) => remainsExpectation(world, condition.val, value)
    )
  ];
}

export async function processExpectationEvent(world: World, event: Event, from: string | null): Promise<World> {
  return await processCommandEvent<any>("Expectation", expectationCommands(), world, event, from);
}

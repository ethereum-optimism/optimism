import {Event} from '../Event';
import {addInvariant, World} from '../World';
import {
  EventV,
  Value
} from '../Value';
import {
  getCoreValue,
  getEventV,
} from '../CoreValue';
import {Invariant} from '../Invariant';
import {StaticInvariant} from '../Invariant/StaticInvariant';
import {RemainsInvariant} from '../Invariant/RemainsInvariant';
import {SuccessInvariant} from '../Invariant/SuccessInvariant';
import {formatEvent} from '../Formatter';
import {Arg, View, processCommandEvent} from '../Command';


async function staticInvariant(world: World, condition): Promise<World> {
  const currentValue = await getCoreValue(world, condition);
  const invariant = new StaticInvariant(condition, currentValue);

  return addInvariant(world, invariant);
}

async function remainsInvariant(world: World, condition: Event, value: Value): Promise<World> {
  const invariant = new RemainsInvariant(condition, value);

  // Immediately check value matches
  await invariant.checker(world, true);

  return addInvariant(world, invariant);
}

async function successInvariant(world: World): Promise<World> {
  const invariant = new SuccessInvariant();

  return addInvariant(world, invariant);
}

export function invariantCommands() {
  return [
    new View<{condition: EventV}>(`
        #### Static

        * "Static <Condition>" - Ensures that the given condition retains a consistent value
          * E.g ."Invariant Static (CToken cZRX UnderlyingBalance Geoff)"
      `,
      "Static",
      [
        new Arg("condition", getEventV)
      ],
      (world, {condition}) => staticInvariant(world, condition.val)
    ),
    new View<{condition: EventV, value: Value}>(`
        #### Remains

        * "Invariant Remains <Condition> <Value>" - Ensures that the given condition starts at and remains a given value
          * E.g ."Invariant Remains (CToken cZRX UnderlyingBalance Geoff) (Exactly 0)"
      `,
      "Remains",
      [
        new Arg("condition", getEventV),
        new Arg("value", getCoreValue)
      ],
      (world, {condition, value}) => remainsInvariant(world, condition.val, value)
    ),
    new View<{}>(`
        #### Success

        * "Invariant Success" - Ensures that each transaction completes successfully
          * E.g ."Invariant Success"
      `,
      "Success",
      [],
      (world, {}) => successInvariant(world)
    )
  ];
}

export async function processInvariantEvent(world: World, event: Event, from: string | null): Promise<World> {
  return await processCommandEvent<any>("Invariant", invariantCommands(), world, event, from);
}

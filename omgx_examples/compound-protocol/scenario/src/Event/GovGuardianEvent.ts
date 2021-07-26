import { Event } from '../Event';
import { addAction, describeUser, World } from '../World';
import { Governor } from '../Contract/Governor';
import { invoke } from '../Invokation';
import {
  getAddressV,
  getEventV,
  getNumberV,
  getStringV,
  getCoreValue
} from '../CoreValue';
import {
  AddressV,
  EventV,
  NumberV,
  StringV
} from '../Value';
import { Arg, Command, processCommandEvent, View } from '../Command';

export function guardianCommands(governor: Governor) {
  return [
    new Command<{ newPendingAdmin: AddressV, eta: NumberV }>(
      `
        #### QueueSetTimelockPendingAdmin

        * "Governor <Governor> QueueSetTimelockPendingAdmin newPendingAdmin:<Address> eta:<Number>" - Queues in the timelock a function to set a new pending admin
        * E.g. "Governor GovernorScenario Guardian QueueSetTimelockPendingAdmin Geoff 604900"
    `,
      'QueueSetTimelockPendingAdmin',
      [
        new Arg('newPendingAdmin', getAddressV),
        new Arg('eta', getNumberV)
      ],
      async (world, from, { newPendingAdmin, eta }) => {
        const invokation = await invoke(world, governor.methods.__queueSetTimelockPendingAdmin(newPendingAdmin.val, eta.encode()), from);

        return addAction(
          world,
          `Gov Guardian has queued in the timelock a new pending admin command for ${describeUser(world, newPendingAdmin.val)}`,
          invokation
        )
      }
    ),

    new Command<{ newPendingAdmin: AddressV, eta: NumberV }>(
      `
        #### ExecuteSetTimelockPendingAdmin

        * "Governor <Governor> ExecuteSetTimelockPendingAdmin newPendingAdmin:<Address> eta:<Number>" - Executes on the timelock the function to set a new pending admin
        * E.g. "Governor GovernorScenario Guardian ExecuteSetTimelockPendingAdmin Geoff 604900"
    `,
      'ExecuteSetTimelockPendingAdmin',
      [
        new Arg('newPendingAdmin', getAddressV),
        new Arg('eta', getNumberV)
      ],
      async (world, from, { newPendingAdmin, eta }) => {
        const invokation = await invoke(world, governor.methods.__executeSetTimelockPendingAdmin(newPendingAdmin.val, eta.encode()), from);

        return addAction(
          world,
          `Gov Guardian has executed via the timelock a new pending admin to ${describeUser(world, newPendingAdmin.val)}`,
          invokation
        )
      }
    ),

    new Command<{}>(
      `
        #### AcceptAdmin

        * "Governor <Governor> Guardian AcceptAdmin" - Calls \`acceptAdmin\` on the Timelock
        * E.g. "Governor GovernorScenario Guardian AcceptAdmin"
    `,
      'AcceptAdmin',
      [],
      async (world, from, { }) => {
        const invokation = await invoke(world, governor.methods.__acceptAdmin(), from);

        return addAction(
          world,
          `Gov Guardian has accepted admin`,
          invokation
        )
      }
    ),

    new Command<{}>(
      `
        #### Abdicate

        * "Governor <Governor> Guardian Abdicate" - Abdicates gov guardian role
        * E.g. "Governor GovernorScenario Guardian Abdicate"
    `,
      'Abdicate',
      [],
      async (world, from, { }) => {
        const invokation = await invoke(world, governor.methods.__abdicate(), from);

        return addAction(
          world,
          `Gov Guardian has abdicated`,
          invokation
        )
      }
    )
  ];
}

export async function processGuardianEvent(world: World, governor: Governor, event: Event, from: string | null): Promise<World> {
  return await processCommandEvent<any>('Guardian', guardianCommands(governor), world, event, from);
}

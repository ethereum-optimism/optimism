import {World} from './World';
import {parse} from './Parser';
import {expandEvent, Macros} from './Macro';
import {processEvents} from './CoreEvent'

export async function runCommand(world: World, command: string, macros: Macros): Promise<World> {
  const trimmedCommand = command.trim();

  const event = parse(trimmedCommand, {startRule: 'step'});

  if (event === null) {
    return world;
  } else {
    world.printer.printLine(`Command: ${trimmedCommand}`);

    let expanded = expandEvent(macros, event);

    return processEvents(world, expanded);
  }
}

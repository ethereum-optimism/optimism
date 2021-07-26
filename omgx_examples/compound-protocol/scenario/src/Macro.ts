import {Event} from './Event';

interface Arg {
  arg: any
  def: any
  splat: any
}

interface Macro {
  args: Arg[]
  steps: Event
}

type ArgMap = {[arg: string]: Event};
type NamedArg = { argName: string, argValue: Event };
type ArgValue = Event | NamedArg;

export type Macros = {[eventName: string]: Macro};

export function expandEvent(macros: Macros, event: Event): Event[] {
  const [eventName, ...eventArgs] = event;

  if (macros[<string>eventName]) {
    let expanded = expandMacro(macros[<string>eventName], eventArgs);

    // Recursively expand steps
    return expanded.map(event => expandEvent(macros, event)).flat();
  } else {
    return [event];
  }
}

function getArgValues(eventArgs: ArgValue[], macroArgs: Arg[]): ArgMap {
  const eventArgNameMap: ArgMap = {};
  const eventArgIndexed: Event[] = [];
  const argValues: ArgMap = {};
  let usedNamedArg: boolean = false;
  let usedSplat: boolean = false;

  eventArgs.forEach((eventArg) => {
    if (eventArg.hasOwnProperty('argName')) {
      const {argName, argValue} = <NamedArg>eventArg;

      eventArgNameMap[argName] = argValue;
      usedNamedArg = true;
    } else {
      if (usedNamedArg) {
        throw new Error("Cannot use positional arg after named arg in macro invokation.");
      }

      eventArgIndexed.push(<Event>eventArg);
    }
  });

  macroArgs.forEach(({arg, def, splat}, argIndex) => {
    let val;

    if (usedSplat) {
      throw new Error("Cannot have arg after splat arg");
    }

    if (eventArgNameMap[arg] !== undefined) {
      val = eventArgNameMap[arg];
    } else if (splat) {
      val = eventArgIndexed.slice(argIndex);
      usedSplat = true;
    } else if (eventArgIndexed[argIndex] !== undefined) {
      val = eventArgIndexed[argIndex];
    } else if (def !== undefined) {
      val = def;
    } else {
      throw new Error("Macro cannot find arg value for " + arg);
    }
    argValues[arg] = val;
  });

  return argValues;
}

export function expandMacro(macro: Macro, event: Event): Event[] {
  const argValues = getArgValues(<ArgValue[]>event, macro.args);

  function expandStep(step) {
    return step.map((token) => {
      if (argValues[token] !== undefined) {
        return argValues[token];
      } else {
        if (Array.isArray(token)) {
          return expandStep(token);
        } else {
          return token;
        }
      }
    });
  };

  return macro.steps.map(expandStep);
}

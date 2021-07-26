import {Value} from './Value';
import {Action} from './Action';
import {EventProcessingError} from './CoreEvent'
import {formatEvent} from './Formatter';

import * as readline from 'readline';

export interface Printer {
  printLine(str: string): void
  printMarkdown(str: string): void
  printValue(val: Value): void
  printError(err: Error): void
  printAction(action: Action<any>): void
}

export class CallbackPrinter implements Printer {
  callback: (message: any, format: object) => void

  constructor(callback: (message: string) => void) {
    this.callback = callback;
  }

  printLine(str: string): void {
    this.callback(str, {});
  }

  printMarkdown(str: string): void {
    this.callback(str, {markdown: true});
  }

  printValue(val: Value): void {
    this.callback(val.toString(), {value: true});
  }

  printError(err: Error): void {
    if (process.env['verbose']) {
      this.callback(err, {error: true});
    }

    this.callback(`Error: ${err.toString()}`, {error: true});
  }

  printAction(action: Action<any>): void {
    // Do nothing
  }
}

export class ConsolePrinter implements Printer {
  verbose: boolean

  constructor(verbose: boolean) {
    this.verbose = verbose;
  }

  printLine(str: string): void {
    console.log(str);
  }

  printMarkdown(str: string): void {
    console.log(str);
  }

  printValue(val: Value): void {
    console.log(val.toString());
  }

  printError(err: Error): void {
    if (this.verbose) {
      console.log(err);
    }

    console.log(`Error: ${err.toString()}`);
  }

  printAction(action: Action<any>): void {
    if (this.verbose) {
      console.log(`Action: ${action.log}`);
    }
  }
}

export class ReplPrinter implements Printer {
  rl : readline.Interface;
  verbose : boolean

  constructor(rl: readline.Interface, verbose: boolean) {
    this.rl = rl;
    this.verbose = verbose;
  }

  printLine(str: string): void {
    console.log(`${str}`);
  }

  printMarkdown(str: string): void {
    console.log(`${str}`);
  }

  printValue(val: Value): void {
    console.log(val.toString());
  }

  printError(err: Error): void {
    if (this.verbose) {
      console.log(err);
    }

    if (err instanceof EventProcessingError) {
      console.log(`Event Processing Error:`);
      console.log(`\t${err.error.toString()}`);
      console.log(`\twhen processing event \`${formatEvent(err.event)}\``);
    } else {
      console.log(`Error: ${err.toString()}`);
    }
  }

  printAction(action: Action<any>): void {
    console.log(`Action: ${action.log}`);
  }
}

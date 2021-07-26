import {Invokation} from './Invokation';

export class Action<T> {
  log: string;
  invokation: Invokation<T>;

  constructor(log: string, invokation: Invokation<T>) {
    this.log = log;
    this.invokation = invokation;
  }

  toString() {
    return `Action: log=${this.log}, result=${this.invokation.toString()}`;
  }
}

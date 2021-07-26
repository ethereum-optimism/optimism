import {World} from './World';
import {Event} from './Event';
import BigNumber from 'bignumber.js';
import {toEncodableNum} from './Encoding';
import {formatEvent} from './Formatter';

BigNumber.config({ ROUNDING_MODE: 3 });
const mantissaOne = new BigNumber('1.0e18');

export enum Order {
  EQUAL,
  LESS_THAN,
  GREATER_THAN
}

export interface Value {
  compareTo(world: World, given: Value): boolean
  compareOrder(world: World, given: Value): Order
  toString(): string
  truthy(): boolean
}

function compareInt(a: number, b: number): Order {
  if (a === b) {
    return Order.EQUAL;
  } else if (a > b) {
    return Order.GREATER_THAN;
  } else {
    return Order.LESS_THAN;
  }
}

export class EventV implements Value {
  val: Event

  constructor(val) {
    this.val = val;
  }

  compareTo(world: World, given: Value): boolean {
    if (given instanceof EventV) {
      return JSON.stringify(this.val) === JSON.stringify(given.val);
    } else {
      throw new Error(`Cannot compare ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
    }
  }

  compareOrder(world: World, given: Value): Order {
    throw new Error(`Cannot compare order of ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
  }

  toString() {
    return `EventV<val=${formatEvent(this.val)}>`;
  }

  truthy() {
    // This check applies to strings or arrays :)
    return this.val.length > 0;
  }
}

export class AnythingV implements Value {
  compareTo(world: World, given: Value): boolean {
    // Everything is awesome.
    return true;
  }

  compareOrder(world: World, given: Value): Order {
    throw new Error(`Cannot compare order of ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
  }

  toString() {
    return `AnythingV<>`;
  }

  truthy() {
    return true;
  }
}

export class NothingV implements Value {
  val: null

  constructor() {
    this.val = null;
  }

  compareTo(world: World, given: Value): boolean {
    // Everything is not awesome.
    return false;
  }

  compareOrder(world: World, given: Value): Order {
    throw new Error(`Cannot compare order of ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
  }

  encode() {
    return null;
  }

  toString() {
    return `NothingV<>`;
  }

  truthy() {
    return false;
  }
}

export class BoolV implements Value {
  val: boolean

  constructor(val) {
    this.val = val;
  }

  compareTo(world: World, given: Value): boolean {
    if (given instanceof BoolV) {
      return this.val === given.val;
    } else if (given instanceof NumberV) {
      return this.compareTo(world, given.toBoolV());
    } else if (given instanceof StringV && ( given.val === 'true' || given.val === 'false' )) {
      return this.val || given.val !== 'true';
    } else {
      throw new Error(`Cannot compare ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
    }
  }

  compareOrder(world: World, given: Value): Order {
    throw new Error(`Cannot compare order of ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
  }

  toString() {
    return `BoolV<val=${this.val}>`;
  }

  truthy() {
    return this.val;
  }
}

export class StringV implements Value {
  val: string

  constructor(val) {
    this.val = val;
  }

  compareTo(world: World, given: Value): boolean {
    if (given instanceof StringV) {
      return this.val === given.val;
    } else if ( given instanceof AddressV) {
      return world.web3.utils.toChecksumAddress(this.val) === world.web3.utils.toChecksumAddress(given.val);
    } else {
      throw new Error(`Cannot compare ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
    }
  }

  compareOrder(world: World, given: Value): Order {
    throw new Error(`Cannot compare order of ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
  }

  toString() {
    return `StringV<val=${this.val}>`;
  }

  truthy() {
    return this.val.length > 0;
  }
}

export class MapV implements Value {
  val: object

  constructor(val) {
    this.val = val;
  }

  compareTo(world: World, given: Value): boolean {
    if (given instanceof MapV) {
      return JSON.stringify(this.val) === JSON.stringify(given.val);
    } else {
      throw new Error(`Cannot compare ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
    }
  }

  compareOrder(world: World, given: Value): Order {
    throw new Error(`Cannot compare order of ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
  }

  toString() {
    return `MapV<val=${JSON.stringify(this.val)}>`;
  }

  truthy() {
    return Object.keys(this.val).length > 0;
  }
}

export class AddressV implements Value {
  val: string

  constructor(val) {
    this.val = val;
  }

  compareTo(world: World, given: Value): boolean {
    if (given instanceof AddressV || given instanceof StringV) {
      return world.web3.utils.toChecksumAddress(this.val) === world.web3.utils.toChecksumAddress(given.val);
    } else {
      throw new Error(`Cannot compare ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
    }
  }

  compareOrder(world: World, given: Value): Order {
    throw new Error(`Cannot compare order of ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
  }

  toString() {
    return `AddressV<val=${this.val}>`;
  }

  truthy() {
    return this.val !== "0x0000000000000000000000000000000000000000";
  }
}

export class NumberV implements Value {
  val : number | string

  constructor(val: number | string, denom?: number | undefined) {
    if (denom) {
      this.val = Number(val) / denom;
    } else {
      this.val = val;
    }
  }

  toNumber(): number {
    return Number(this.val);
  }

  encode() {
    return toEncodableNum(this.val);
  }

  compareTo(world: World, given: Value): boolean {
    if (given instanceof NumberV) {
      const thisBig = new BigNumber(this.val).toFixed();
      const givenBig = new BigNumber(given.val).toFixed();

      return thisBig === givenBig;
    } else if (given instanceof PreciseV) {
      return this.compareTo(world, given.toNumberV());
    } else if (given instanceof StringV) {
      return this.compareTo(world, new NumberV(Number(given.val)));
    } else {
      throw new Error(`Cannot compare ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
    }
  }

  compareOrder(world: World, given: Value): Order {
    if (given instanceof NumberV) {
      const thisBig = new BigNumber(this.val).toNumber();
      const givenBig = new BigNumber(given.val).toNumber();

      return compareInt(thisBig, givenBig);
    } else if (given instanceof PreciseV) {
      return this.compareOrder(world, given.toNumberV());
    } else {
      throw new Error(`Cannot compare ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
    }
  }

  toBoolV(): BoolV {
    if (this.val === 0) {
      return new BoolV(true)
    } else if (this.val === 1) {
      return new BoolV(false);
    }

    throw new Error(`Cannot convert number ${this.val} into bool`)
  }

  asExp(denom=undefined): string {
    return new BigNumber(this.val).toExponential();
  }

  show(): string {
    return new BigNumber(this.val).toExponential();
  }

  toString() {
    return `NumberV<val=${this.val},exp=${this.asExp()}>`;
  }

  truthy() {
    return this.val != 0;
  }

  add(b: NumberV): NumberV {
    return new NumberV(new BigNumber(this.val).plus(new BigNumber(b.val)).toFixed());
  }

  div(b: NumberV): NumberV {
    return new NumberV(new BigNumber(this.val).div(new BigNumber(b.val)).toFixed());
  }

  mul(b: NumberV): NumberV {
    return new NumberV(new BigNumber(this.val).times(new BigNumber(b.val)).toFixed());
  }

  sub(b: NumberV): NumberV {
    return new NumberV(new BigNumber(this.val).minus(new BigNumber(b.val)).toFixed());
  }
}

export class ExpNumberV extends NumberV {
  show() {
    return new BigNumber(this.val).dividedBy(mantissaOne).toNumber().toString();
  }
}

export class PercentV extends NumberV {
  show() {
    return new BigNumber(this.val).dividedBy(mantissaOne).multipliedBy(new BigNumber(100)).toNumber().toString() + '%';
  }
}

export class PreciseV implements Value {
  val: number
  precision: number

  constructor(val, precision) {
    this.val = val;
    this.precision = precision;
  }

  compareTo(world: World, given: Value): boolean {
    if (given instanceof NumberV) {
      const thisBig = new BigNumber(this.val.toString()).toPrecision(this.precision);
      const givenBig = new BigNumber(given.val.toString()).toPrecision(this.precision);

      return thisBig === givenBig;
    } else if (given instanceof PreciseV) {
      // TODO: Is this okay?
      return this.compareTo(world, given.toNumberV());
    } else {
      throw new Error(`Cannot compare ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
    }
  }

  compareOrder(world: World, given: Value): Order {
    throw new Error(`Cannot compare order of ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
  }

  toNumberV(): NumberV {
    return new NumberV(this.val);
  }

  toString() {
    return `PreciseV<val=${this.val}, precision=${this.precision}>`;
  }

  truthy() {
    return this.val != 0;
  }
}

export class ListV implements Value {
  val: Value[]

  constructor(els) {
    this.val = els;
  }

  compareTo(world: World, given: Value): boolean {
    if (given instanceof ListV || given instanceof ArrayV) {
      return this.val.every((el, i) => el.compareTo(world, given.val[i] || new NothingV()));
    } else {
      throw new Error(`Cannot compare ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
    }
  }

  compareOrder(world: World, given: Value): Order {
    throw new Error(`Cannot compare order of ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
  }

  toString() {
    return `ListV<val=${this.val.map(el => el.toString()).join(',')}>`;
  }

  truthy() {
    return this.val.length > 0;
  }
}

export class ArrayV<T extends Value> implements Value {
  val: T[]

  constructor(els) {
    this.val = els;
  }

  compareTo(world: World, given: Value): boolean {
    if (given instanceof ListV || given instanceof ArrayV) {
      return this.val.every((el, i) => el.compareTo(world, given.val[i] || new NothingV()));
    } else {
      throw new Error(`Cannot compare ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
    }
  }

  compareOrder(world: World, given: Value): Order {
    throw new Error(`Cannot compare order of ${typeof this} to ${typeof given} (${this.toString()}, ${given.toString()})`);
  }

  toString() {
    return `ArrayV<val=${this.val.map(el => el.toString()).join(',')}>`;
  }

  truthy() {
    return this.val.length > 0;
  }
}

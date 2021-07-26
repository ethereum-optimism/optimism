import {Invariant} from '../Invariant';
import {fail, World} from '../World';
import {getCoreValue} from '../CoreValue';
import {Value} from '../Value';
import {Event} from '../Event';
import {formatEvent} from '../Formatter';

export class StaticInvariant implements Invariant {
  condition: Event;
  value: Value;
  held = false;

  constructor(condition: Event, value: Value) {
    this.condition = condition;
    this.value = value;
  }

  async getCurrentValue(world: World): Promise<Value> {
    return await getCoreValue(world, this.condition);
  };

  async checker(world: World): Promise<void> {
    const currentValue = await this.getCurrentValue(world);

    if (!this.value.compareTo(world, currentValue)) {
      fail(world, `Static invariant broken! Expected ${this.toString()} to remain static value \`${this.value}\` but became \`${currentValue}\``);
    }
  }

  toString() {
    return `StaticInvariant: condition=${formatEvent(this.condition)}, value=${this.value.toString()}`;
  }
}

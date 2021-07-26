
type ScalarEvent = string;
interface EventArray extends Array<ScalarEvent | EventArray> {
    [index: number]: ScalarEvent | EventArray;
}

export type Event = EventArray;

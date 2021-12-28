/**
 * Utility type for deep partials.
 */
export type DeepPartial<T> = {
  [P in keyof T]?: DeepPartial<T[P]>
}

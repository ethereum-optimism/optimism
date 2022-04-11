// TODO: A lot of this stuff could probably live in core-utils instead.
// Review this file eventually for stuff that could go into core-utils.

/**
 * Returns a copy of the given object ({ ...obj }) with the given keys omitted.
 *
 * @param obj Object to return with the keys omitted.
 * @param keys Keys to omit from the returned object.
 * @returns A copy of the given object with the given keys omitted.
 */
export const omit = <T extends object, K extends string | number | symbol>(
  obj: T,
  ...keys: K[]
): Omit<T, K> => {
  const copy = { ...obj }
  for (const key of keys) {
    delete copy[key as string]
  }
  return copy
}

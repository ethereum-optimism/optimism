/**
 * All of the below functions check whether or not the two provided objects are equal,
 * returning true if they are and false otherwise
 */

export const areEqual = (one: any, two: any): boolean => {
  if (
    (one === undefined && two === undefined) ||
    (one === null && two === null) ||
    (one === false && two === false)
  ) {
    return true
  }

  if (Array.isArray(one) || Array.isArray(two)) {
    if (
      Array.isArray(one) !== Array.isArray(two) ||
      one.length !== two.length
    ) {
      return false
    }
    for (let i = 0; i < one.length; i++) {
      if (!areEqual(one[i], two[i])) {
        return false
      }
    }
    return true
  }

  if (typeof one === 'object' && typeof two === 'object') {
    return objectsEqual(one, two)
  }

  return one === two
}

export const objectsEqual = (obj1: {}, obj2: {}): boolean => {
  if (!obj1 && !obj2) {
    return true
  }

  if (!obj1 || !obj2) {
    return false
  }

  if (obj1.hasOwnProperty('equals')) {
    return obj1['equals'](obj2)
  }

  const props: string[] = Object.getOwnPropertyNames(obj1)
  if (props.length !== Object.getOwnPropertyNames(obj2).length) {
    return false
  }

  for (const prop of props) {
    if (!obj2.hasOwnProperty(prop)) {
      return false
    }

    if (typeof obj1[prop] === 'object' && typeof obj2[prop] === 'object') {
      if (objectsEqual(obj1[prop], obj2[prop])) {
        continue
      } else {
        return false
      }
    }

    // TODO: This won't work for reference types, but it'll work for now
    if (obj1[prop] !== obj2[prop]) {
      return false
    }
  }

  return true
}

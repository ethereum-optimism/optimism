import { Decider, Decision, ImplicationProofItem } from '../../../types/ovm'

export class CannotDecideError extends Error {
  constructor(message?: string) {
    super(message)
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export const handleCannotDecideError = (e): undefined => {
  if (!(e instanceof CannotDecideError)) {
    throw e
  }

  return undefined
}

const getJustification = (
  decider: Decider,
  input: any,
  witness: any
): ImplicationProofItem[] => {
  return [
    {
      implication: {
        decider,
        input,
      },
      implicationWitness: witness,
    },
  ]
}

export class TrueDecider implements Decider {
  public async decide(input: any): Promise<Decision> {
    return {
      outcome: true,
      justification: getJustification(this, input, true),
    }
  }
}

export class FalseDecider implements Decider {
  public async decide(input: any): Promise<Decision> {
    return {
      outcome: false,
      justification: getJustification(this, input, true),
    }
  }
}

export class CannotDecideDecider implements Decider {
  public async decide(input: any): Promise<Decision> {
    throw new CannotDecideError('Cannot decide!')
  }
}

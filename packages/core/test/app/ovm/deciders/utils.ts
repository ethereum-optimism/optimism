import {
  Decider,
  Decision,
  ImplicationProofItem,
} from '../../../../src/types/ovm'
import { CannotDecideError } from '../../../../src/app/ovm/deciders'

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

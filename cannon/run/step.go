package run

type StepFn func(proof bool) (*StepWitness, error)

"""
Dispute Game State Transition Proof Validator

This module provides a pure Python implementation of a state transition proof validator
for Ethereum Optimism's dispute game, as described in the bounty:
https://github.com/ethereum-optimism/optimism/issues/bounty_889

This implementation is self-contained, does not use external dependencies, and is suitable
for integration into testing or validation frameworks.

Assumptions:
- The state transition proof consists of a sequence of steps, each with a pre-state,
  an input, and a post-state.
- The validator checks that applying the input to the pre-state yields the post-state,
  according to a provided state transition function.
- The state and input are represented as dictionaries with string keys and integer values.
- The state transition function is provided by the user and must be deterministic.

Author: OpenAI's Assistant
"""

from typing import Callable, Dict, List, Any


State = Dict[str, int]
Input = Dict[str, int]


class StateTransitionProofStep:
    """
    Represents a single step in a state transition proof.
    """
    def __init__(self, pre_state: State, input_data: Input, post_state: State) -> None:
        self.pre_state = pre_state
        self.input_data = input_data
        self.post_state = post_state

    def __repr__(self) -> str:
        return (
            f"StateTransitionProofStep(pre_state={self.pre_state}, "
            f"input_data={self.input_data}, post_state={self.post_state})"
        )


class StateTransitionProof:
    """
    Represents a full state transition proof, as a sequence of steps.
    """
    def __init__(self, steps: List[StateTransitionProofStep]) -> None:
        self.steps = steps

    def __repr__(self) -> str:
        return f"StateTransitionProof(steps={self.steps})"


class StateTransitionProofValidator:
    """
    Validates a state transition proof for a dispute game.

    The validator checks that for each step, applying the transition function to the
    pre-state and input yields the post-state.
    """

    def __init__(self, transition_fn: Callable[[State, Input], State]) -> None:
        """
        :param transition_fn: A deterministic function that takes a state and input,
                              and returns the next state.
        """
        self.transition_fn = transition_fn

    def validate(self, proof: StateTransitionProof) -> bool:
        """
        Validates the given state transition proof.

        :param proof: The state transition proof to validate.
        :return: True if the proof is valid, False otherwise.
        """
        for idx, step in enumerate(proof.steps):
            expected_post_state = self.transition_fn(step.pre_state, step.input_data)
            if not self._states_equal(expected_post_state, step.post_state):
                print(
                    f"Validation failed at step {idx}:\n"
                    f"  Pre-state: {step.pre_state}\n"
                    f"  Input: {step.input_data}\n"
                    f"  Expected post-state: {expected_post_state}\n"
                    f"  Actual post-state:   {step.post_state}"
                )
                return False
        return True

    @staticmethod
    def _states_equal(state1: State, state2: State) -> bool:
        """
        Compares two states for equality.

        :param state1: First state.
        :param state2: Second state.
        :return: True if states are equal, False otherwise.
        """
        return state1 == state2


# Example transition function for demonstration purposes.
def example_transition_fn(state: State, input_data: Input) -> State:
    """
    Example state transition function: adds input values to state values.

    :param state: Current state.
    :param input_data: Input to apply.
    :return: Next state.
    """
    next_state = state.copy()
    for key, value in input_data.items():
        next_state[key] = next_state.get(key, 0) + value
    return next_state


# Example test cases
def _run_tests() -> None:
    """
    Runs basic tests for the StateTransitionProofValidator.
    """
    # Step 1: state = {'a': 1}, input = {'a': 2}, post_state = {'a': 3}
    step1 = StateTransitionProofStep(
        pre_state={'a': 1},
        input_data={'a': 2},
        post_state={'a': 3}
    )
    # Step 2: state = {'a': 3}, input = {'b': 4}, post_state = {'a': 3, 'b': 4}
    step2 = StateTransitionProofStep(
        pre_state={'a': 3},
        input_data={'b': 4},
        post_state={'a': 3, 'b': 4}
    )
    proof = StateTransitionProof([step1, step2])
    validator = StateTransitionProofValidator(example_transition_fn)
    assert validator.validate(proof), "Proof should be valid"

    # Invalid proof: wrong post_state
    bad_step = StateTransitionProofStep(
        pre_state={'a': 1},
        input_data={'a': 2},
        post_state={'a': 4}  # Should be 3
    )
    bad_proof = StateTransitionProof([bad_step])
    assert not validator.validate(bad_proof), "Proof should be invalid"

    print("All tests passed.")


if __name__ == "__main__":
    _run_tests()
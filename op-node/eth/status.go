package eth

import (
	"fmt"
)

func ForkchoiceUpdateErr(payloadStatus PayloadStatusV1) error {
	switch payloadStatus.Status {
	case ExecutionSyncing:
		return fmt.Errorf("updated forkchoice, but node is syncing")
	case ExecutionAccepted, ExecutionInvalidTerminalBlock, ExecutionInvalidBlockHash:
		// ACCEPTED, INVALID_TERMINAL_BLOCK, INVALID_BLOCK_HASH are only for execution
		return fmt.Errorf("unexpected %s status, could not update forkchoice", payloadStatus.Status)
	case ExecutionInvalid:
		return fmt.Errorf("cannot update forkchoice, block is invalid")
	case ExecutionValid:
		return nil
	default:
		return fmt.Errorf("unknown forkchoice status: %q", string(payloadStatus.Status))
	}
}

func NewPayloadErr(payload *ExecutionPayload, payloadStatus *PayloadStatusV1) error {
	switch payloadStatus.Status {
	case ExecutionValid:
		return nil
	case ExecutionSyncing:
		return fmt.Errorf("failed to execute payload %s, node is syncing", payload.ID())
	case ExecutionInvalid:
		return fmt.Errorf("execution payload %s was INVALID! Latest valid hash is %s, ignoring bad block: %v", payload.ID(), payloadStatus.LatestValidHash, payloadStatus.ValidationError)
	case ExecutionInvalidBlockHash:
		return fmt.Errorf("execution payload %s has INVALID BLOCKHASH! %v", payload.BlockHash, payloadStatus.ValidationError)
	case ExecutionInvalidTerminalBlock:
		return fmt.Errorf("engine is misconfigured. Received invalid-terminal-block error while engine API should be active at genesis. err: %v", payloadStatus.ValidationError)
	case ExecutionAccepted:
		return fmt.Errorf("execution payload cannot be validated yet, latest valid hash is %s", payloadStatus.LatestValidHash)
	default:
		return fmt.Errorf("unknown execution status on %s: %q, ", payload.ID(), string(payloadStatus.Status))
	}
}

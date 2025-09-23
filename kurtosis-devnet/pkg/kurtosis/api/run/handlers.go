package run

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/fatih/color"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Color printers
var (
	printCyan   = color.New(color.FgCyan).SprintFunc()
	printYellow = color.New(color.FgYellow).SprintFunc()
	printRed    = color.New(color.FgRed).SprintFunc()
	printBlue   = color.New(color.FgBlue).SprintFunc()
)

// MessageHandler defines the interface for handling different types of messages
type MessageHandler interface {
	// Handle processes the message if applicable and returns:
	// - bool: whether the message was handled
	// - error: any error that occurred during handling
	Handle(context.Context, interfaces.StarlarkResponse) (bool, error)
}

// MessageHandlerFunc is a function type that implements MessageHandler
type MessageHandlerFunc func(context.Context, interfaces.StarlarkResponse) (bool, error)

func (f MessageHandlerFunc) Handle(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
	return f(ctx, resp)
}

// FirstMatchHandler returns a handler that applies the first matching handler from the given handlers
func FirstMatchHandler(handlers ...MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
		for _, h := range handlers {
			handled, err := h.Handle(ctx, resp)
			if err != nil {
				return true, err
			}
			if handled {
				return true, nil
			}
		}
		return false, nil
	})
}

// AllHandlers returns a handler that applies all the given handlers in order
func AllHandlers(handlers ...MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
		anyHandled := false
		for _, h := range handlers {
			handled, err := h.Handle(ctx, resp)
			if err != nil {
				return true, err
			}
			anyHandled = anyHandled || handled
		}
		return anyHandled, nil
	})
}

// defaultHandler is the default message handler that provides standard Kurtosis output
type defaultHandler struct {
	tracer trace.Tracer
	span   trace.Span
}

func newDefaultHandler() *defaultHandler {
	return &defaultHandler{
		tracer: otel.Tracer("kurtosis-run"),
	}
}

var _ MessageHandler = (*defaultHandler)(nil)

func (h *defaultHandler) Handle(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
	hdlr := FirstMatchHandler(
		MessageHandlerFunc(h.handleProgress),
		MessageHandlerFunc(h.handleInstruction),
		MessageHandlerFunc(h.handleWarning),
		MessageHandlerFunc(h.handleInfo),
		MessageHandlerFunc(h.handleResult),
		MessageHandlerFunc(h.handleError),
	)

	return hdlr.Handle(ctx, resp)
}

// handleProgress handles progress info messages
func (h *defaultHandler) handleProgress(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
	if progressInfo := resp.GetProgressInfo(); progressInfo != nil {
		// ignore progress messages, same as kurtosis run does
		return true, nil
	}
	return false, nil
}

// handleInstruction handles instruction messages
func (h *defaultHandler) handleInstruction(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
	if instruction := resp.GetInstruction(); instruction != nil {
		desc := instruction.GetDescription()
		_, span := h.tracer.Start(ctx, desc)
		h.span = span

		fmt.Println(printCyan(desc))
		return true, nil
	}
	return false, nil
}

// handleWarning handles warning messages
func (h *defaultHandler) handleWarning(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
	if warning := resp.GetWarning(); warning != nil {
		fmt.Println(printYellow(warning.GetMessage()))
		return true, nil
	}
	return false, nil
}

// handleInfo handles info messages
func (h *defaultHandler) handleInfo(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
	if info := resp.GetInfo(); info != nil {
		fmt.Println(printBlue(info.GetMessage()))
		return true, nil
	}
	return false, nil
}

// handleResult handles instruction result messages
func (h *defaultHandler) handleResult(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
	if result := resp.GetInstructionResult(); result != nil {
		if result.GetSerializedInstructionResult() != "" {
			fmt.Printf("%s\n\n", result.GetSerializedInstructionResult())
		}
		if h.span != nil {
			h.span.End()
		}
		return true, nil
	}
	return false, nil
}

// handleError handles error messages
func (h *defaultHandler) handleError(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
	if err := resp.GetError(); err != nil {
		if interpretErr := err.GetInterpretationError(); interpretErr != nil {
			return true, fmt.Errorf(printRed("interpretation error: %v"), interpretErr)
		}
		if validationErr := err.GetValidationError(); validationErr != nil {
			return true, fmt.Errorf(printRed("validation error: %v"), validationErr)
		}
		if executionErr := err.GetExecutionError(); executionErr != nil {
			return true, fmt.Errorf(printRed("execution error: %v"), executionErr)
		}
		return true, nil
	}
	return false, nil
}

// makeRunFinishedHandler creates a handler for run finished events
func makeRunFinishedHandler(isSuccessful *bool) MessageHandlerFunc {
	return func(ctx context.Context, resp interfaces.StarlarkResponse) (bool, error) {
		if event := resp.GetRunFinishedEvent(); event != nil {
			*isSuccessful = event.GetIsRunSuccessful()
			return true, nil
		}
		return false, nil
	}
}

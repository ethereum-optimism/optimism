package logmods

import (
	"log/slog"
)

// HandlerMod is a log-handler wrapping function.
// Loggers may compose handlers.
// This returns a log handler that may be unwrapped, through the Handler extension interface.
type HandlerMod func(slog.Handler) slog.Handler

// Handler is a slog.Handler you can unwrap,
// to access inner handler functionality.
type Handler interface {
	slog.Handler
	Unwrap() slog.Handler
}

// FindHandler finds a handler with a particular handler type, or returns ok=false if not found.
func FindHandler[H slog.Handler](h slog.Handler) (out H, ok bool) {
	for {
		if h == nil {
			ok = false
			return // zero/nil out value
		}
		// Check if we found our handler
		if found, tempOk := h.(H); tempOk {
			return found, true
		}
		// continue to unwrap if we can
		unwrappable, tempOk := h.(Handler)
		if !tempOk {
			ok = false
			return // zero/nil out value
		}
		h = unwrappable.Unwrap()
	}
}

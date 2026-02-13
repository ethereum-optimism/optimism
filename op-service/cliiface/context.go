package cliiface

import (
	"time"
)

// Context defines the minimal surface of urfave/cli.Context that our
// configuration constructors depend on, across services.
type Context interface {
	String(name string) string
	Bool(name string) bool
	Int(name string) int
	Uint(name string) uint
	Uint64(name string) uint64
	Float64(name string) float64
	Duration(name string) time.Duration
	StringSlice(name string) []string
	IsSet(name string) bool
	Path(name string) string
	Generic(name string) any
}

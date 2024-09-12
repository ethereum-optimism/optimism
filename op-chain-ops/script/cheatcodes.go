package script

// CheatCodesPrecompile implements the Forge vm cheatcodes.
// Note that forge-std wraps these cheatcodes,
// and provides additional convenience functions that use these cheatcodes.
type CheatCodesPrecompile struct {
	h *Host
}

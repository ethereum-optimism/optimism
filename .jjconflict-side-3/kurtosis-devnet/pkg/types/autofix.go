package types

type AutofixMode string

const (
	AutofixModeDisabled AutofixMode = "disabled"
	AutofixModeNormal   AutofixMode = "normal"
	AutofixModeNuke     AutofixMode = "nuke"
)

package flags

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

// VirtualCLI wraps a base CLI context and returns namespaced values for flags
// and supports overriding values for specific flags
type VirtualCLI struct {
	inner   *cli.Context
	chainID uint64
	// overrides allow for the apparent CLI context to be overridden with custom values
	// this is useful for enforcing defaults or disabling features which aren't supported by the virtual node
	stringOverrides map[string]string
	boolOverrides   map[string]bool
}

func NewVirtualCLI(base *cli.Context, chainID uint64) *VirtualCLI {
	return &VirtualCLI{inner: base, chainID: chainID, stringOverrides: make(map[string]string), boolOverrides: make(map[string]bool)}
}

func (v *VirtualCLI) chainName(name string) string {
	return fmt.Sprintf("%s%d.%s", VNFlagNamePrefix, v.chainID, name)
}

func (v *VirtualCLI) globalName(name string) string {
	return VNFlagGlobalPrefix + name
}

// IsSet satisfies the cliiface.Context interface
// if an override is set, or if any namespaced version is set, return true
// otherwise defer to the inner context
func (v *VirtualCLI) IsSet(name string) bool {
	if _, ok := v.stringOverrides[name]; ok {
		return true
	}
	if _, ok := v.boolOverrides[name]; ok {
		return true
	}
	if v.inner.IsSet(v.chainName(name)) {
		return true
	}
	return v.inner.IsSet(v.globalName(name))
}

func (v *VirtualCLI) String(name string) string {
	if val, ok := v.stringOverrides[name]; ok {
		return val
	}
	cName := v.chainName(name)
	if s := v.inner.String(cName); s != "" {
		return s
	}
	gName := v.globalName(name)
	if s := v.inner.String(gName); s != "" {
		return s
	}
	// if there is generic content, return it as a string
	if g := v.inner.Generic(gName); g != nil {
		return fmt.Sprintf("%v", g)
	}
	return ""
}

func (v *VirtualCLI) Bool(name string) bool {
	if val, ok := v.boolOverrides[name]; ok {
		return val
	}
	cName := v.chainName(name)
	if v.inner.IsSet(cName) {
		return v.inner.Bool(cName)
	}
	gName := v.globalName(name)
	return v.inner.Bool(gName)
}

func (v *VirtualCLI) Int(name string) int {
	cName := v.chainName(name)
	if v.inner.IsSet(cName) {
		return v.inner.Int(cName)
	}
	gName := v.globalName(name)
	return v.inner.Int(gName)
}

func (v *VirtualCLI) Uint(name string) uint {
	cName := v.chainName(name)
	if v.inner.IsSet(cName) {
		return v.inner.Uint(cName)
	}
	gName := v.globalName(name)
	return v.inner.Uint(gName)
}

func (v *VirtualCLI) Uint64(name string) uint64 {
	cName := v.chainName(name)
	if v.inner.IsSet(cName) {
		return v.inner.Uint64(cName)
	}
	gName := v.globalName(name)
	return v.inner.Uint64(gName)
}

func (v *VirtualCLI) Float64(name string) float64 {
	cName := v.chainName(name)
	if v.inner.IsSet(cName) {
		return v.inner.Float64(cName)
	}
	gName := v.globalName(name)
	return v.inner.Float64(gName)
}

func (v *VirtualCLI) Duration(name string) time.Duration {
	cName := v.chainName(name)
	if v.inner.IsSet(cName) {
		return v.inner.Duration(cName)
	}
	gName := v.globalName(name)
	return v.inner.Duration(gName)
}

func (v *VirtualCLI) StringSlice(name string) []string {
	cName := v.chainName(name)
	if v.inner.IsSet(cName) {
		return v.inner.StringSlice(cName)
	}
	gName := v.globalName(name)
	return v.inner.StringSlice(gName)
}

func (v *VirtualCLI) Path(name string) string {
	cName := v.chainName(name)
	if v.inner.IsSet(cName) {
		return v.inner.Path(cName)
	}
	gName := v.globalName(name)
	return v.inner.Path(gName)
}

func (v *VirtualCLI) Generic(name string) any {
	cName := v.chainName(name)
	if v.inner.IsSet(cName) {
		return v.inner.Generic(cName)
	}
	gName := v.globalName(name)
	return v.inner.Generic(gName)
}

// WithStringOverride sets a string override for the given base flag name
func (v *VirtualCLI) WithStringOverride(name, value string) *VirtualCLI {
	v.stringOverrides[name] = value
	return v
}

// WithBoolOverride sets a bool override for the given base flag name
func (v *VirtualCLI) WithBoolOverride(name string, value bool) *VirtualCLI {
	v.boolOverrides[name] = value
	return v
}

// Disable annotating anonymous memory mappings. Cannon doesn't support this syscall
// The directive (and functionality) only exists on go1.25 and above so this file is conditionally included.

//go:build go1.25

//go:debug decoratemappings=0

package main

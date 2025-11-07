//go:debug decoratemappings=0
package main

import (
	"common/syscalltests"
)

func main() {
	syscalltests.EventfdTest()
}

package flags

import (
	"fmt"
	"strings"
)

func EnumString[T fmt.Stringer](values []T) string {
	var out strings.Builder
	for i, v := range values {
		out.WriteString(v.String())
		if i+1 < len(values) {
			out.WriteString(", ")
		}
	}
	return out.String()
}

package validations

import (
	"bytes"
	"fmt"
	"strings"
)

type Output struct {
	Errors []string
}

func (o *Output) AsMarkdown() string {
	buf := new(bytes.Buffer)
	writeMarkdownTableHeader(buf, "Error", "Description")

	if len(o.Errors) == 0 {
		writeMarkdownTableRow(buf, "No errors.", "No errors.")
		return buf.String()
	}

	for _, error := range o.Errors {
		errDesc := ErrorDescription(error)
		if errDesc == "" {
			errDesc = "Unknown error code, please check the implementation for more details."
		}
		writeMarkdownTableRow(buf, error, errDesc)
	}

	return buf.String()
}

func writeMarkdownTableHeader(buf *bytes.Buffer, columns ...string) {
	writeMarkdownTableRow(buf, columns...)
	for i := range columns {
		if i > 0 {
			buf.WriteString("|")
		}
		buf.WriteString("---")
	}
	buf.WriteString("\n")
}

func writeMarkdownTableRow(buf *bytes.Buffer, columns ...string) {
	for i, column := range columns {
		if i > 0 {
			buf.WriteString("|")
		}
		fmt.Fprintf(buf, " %s ", markdownCell(column))
	}
	buf.WriteString("\n")
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", "<br>")
	return value
}

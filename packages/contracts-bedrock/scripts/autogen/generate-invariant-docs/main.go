package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	NatspecInv         = "@custom:invariant"
	BaseInvariantGhUrl = "../test/invariants/"
)

// Contract represents an invariant test contract
type Contract struct {
	Name     string
	FileName string
	Docs     []InvariantDoc
}

// InvariantDoc represents the documentation of an invariant
type InvariantDoc struct {
	Header string
	Desc   string
	LineNo int
}

var writtenFiles []string

// Generate the docs
func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Println("Expected path of contracts-bedrock as CLI argument")
		os.Exit(1)
	}
	rootDir := flag.Arg(0)

	invariantsDir := filepath.Join(rootDir, "test/invariants")
	fmt.Printf("invariants dir: %s\n", invariantsDir)
	docsDir := filepath.Join(rootDir, "invariant-docs")
	fmt.Printf("invariant docs dir: %s\n", docsDir)

	// Forge
	fmt.Println("Generating docs for forge invariants...")
	if err := docGen(invariantsDir, docsDir); err != nil {
		fmt.Printf("Failed to generate invariant docs: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generating table-of-contents...")
	// Generate an updated table of contents
	if err := tocGen(docsDir); err != nil {
		fmt.Printf("Failed to generate TOC of docs: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done!")
}

// Lazy-parses all test files in the `test/invariants` directory
// to generate documentation on all invariant tests.
func docGen(invariantsDir, docsDir string) error {

	// Grab all files within the invariants test dir
	files, err := os.ReadDir(invariantsDir)
	if err != nil {
		return fmt.Errorf("error reading directory: %w", err)
	}

	// Array to store all found invariant documentation comments.
	var docs []Contract

	for _, file := range files {
		// Read the contents of the invariant test file.
		fileName := file.Name()
		filePath := filepath.Join(invariantsDir, fileName)
		fileContents, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("error reading file %q: %w", filePath, err)
		}

		// Split the file into individual lines and trim whitespace.
		lines := strings.Split(string(fileContents), "\n")
		for i, line := range lines {
			lines[i] = strings.TrimSpace(line)
		}

		// Create an object to store all invariant test docs for the current contract
		name := strings.Replace(fileName, ".t.sol", "", 1)
		contract := Contract{Name: name, FileName: fileName}

		var currentDoc InvariantDoc

		// Loop through all lines to find comments.
		for i := 0; i < len(lines); i++ {
			line := lines[i]

			// We have an invariant doc
			if strings.HasPrefix(line, "/// "+NatspecInv) {
				// Assign the header of the invariant doc.
				currentDoc = InvariantDoc{
					Header: strings.TrimSpace(strings.Replace(line, "/// "+NatspecInv, "", 1)),
					Desc:   "",
				}
				i++

				// If the header is multi-line, continue appending to the `currentDoc`'s header.
				for {
					if i >= len(lines) {
						break
					}
					line = lines[i]
					i++
					if !(strings.HasPrefix(line, "///") && strings.TrimSpace(line) != "///") {
						break
					}
					currentDoc.Header += " " + strings.TrimSpace(strings.Replace(line, "///", "", 1))
				}

				// Process the description
				for {
					if i >= len(lines) {
						break
					}
					line = lines[i]
					i++
					if !strings.HasPrefix(line, "///") {
						break
					}
					line = strings.TrimSpace(strings.Replace(line, "///", "", 1))

					// If the line has any contents, insert it into the desc.
					// Otherwise, consider it a linebreak.
					if len(line) > 0 {
						currentDoc.Desc += line + " "
					} else {
						currentDoc.Desc += "\n"
					}
				}

				// Set the line number of the test
				currentDoc.LineNo = i

				// Add the doc to the contract
				contract.Docs = append(contract.Docs, currentDoc)
			}
		}

		// Add the contract to the array of docs
		docs = append(docs, contract)
	}

	for _, contract := range docs {
		filePath := filepath.Join(docsDir, contract.Name+".md")
		alreadyWritten := slices.Contains(writtenFiles, filePath)

		// If the file has already been written, append the extra docs to the end.
		// Otherwise, write the file from scratch.
		var fileContent string
		if alreadyWritten {
			existingContent, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("error reading existing file %q: %w", filePath, err)
			}
			fileContent = string(existingContent) + "\n" + renderContractDoc(contract, false)
		} else {
			fileContent = renderContractDoc(contract, true)
		}

		err = os.WriteFile(filePath, []byte(fileContent), 0644)
		if err != nil {
			return fmt.Errorf("error writing file %q: %w", filePath, err)
		}

		if !alreadyWritten {
			writtenFiles = append(writtenFiles, filePath)
		}
	}

	_, _ = fmt.Fprintf(os.Stderr,
		"Generated invariant test documentation for:\n"+
			" - %d contracts\n"+
			" - %d invariant tests\n"+
			"successfully!\n",
		len(docs),
		func() int {
			total := 0
			for _, contract := range docs {
				total += len(contract.Docs)
			}
			return total
		}(),
	)
	return nil
}

// Generate a table of contents for all invariant docs and place it in the README.
func tocGen(docsDir string) error {
	autoTOCPrefix := "<!-- START autoTOC -->\n"
	autoTOCPostfix := "<!-- END autoTOC -->\n"

	files, err := os.ReadDir(docsDir)
	if err != nil {
		return fmt.Errorf("error reading directory %q: %w", docsDir, err)
	}

	// Generate a table of contents section.
	var tocList []string
	for _, file := range files {
		fileName := file.Name()
		if fileName != "README.md" {
			tocList = append(tocList, fmt.Sprintf("- [%s](./%s)", strings.Replace(fileName, ".md", "", 1), fileName))
		}
	}
	toc := fmt.Sprintf("%s\n## Table of Contents\n%s\n%s",
		autoTOCPrefix, strings.Join(tocList, "\n"), autoTOCPostfix)

	// Write the table of contents to the README.
	readmePath := filepath.Join(docsDir, "README.md")
	readmeContents, err := os.ReadFile(readmePath)
	if err != nil {
		return fmt.Errorf("error reading README file %q: %w", readmePath, err)
	}
	readmeParts := strings.Split(string(readmeContents), autoTOCPrefix)
	above := readmeParts[0]
	readmeParts = strings.Split(readmeParts[1], autoTOCPostfix)
	below := readmeParts[1]
	err = os.WriteFile(readmePath, []byte(above+toc+below), 0644)
	if err != nil {
		return fmt.Errorf("error writing README file %q: %w", readmePath, err)
	}
	return nil
}

// Render a `Contract` object into valid markdown.
func renderContractDoc(contract Contract, header bool) string {
	var sb strings.Builder

	if header {
		sb.WriteString(fmt.Sprintf("# `%s` Invariants\n", contract.Name))
	}
	sb.WriteString("\n")

	for i, doc := range contract.Docs {
		line := fmt.Sprintf("%s#L%d", contract.FileName, doc.LineNo)
		sb.WriteString(fmt.Sprintf("## %s\n**Test:** [`%s`](%s%s)\n\n%s", doc.Header, line, BaseInvariantGhUrl, line, doc.Desc))
		if i != len(contract.Docs)-1 {
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

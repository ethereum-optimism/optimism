/**
 * @fileoverview Go CLI application for generating Kurtosis parameters from an Optimism devnet manifest YAML file.
 * The application ensures secure file handling and correct serialization for the Kurtosis environment.
 */
package main

import (
	"fmt"
	"io" // Used for efficient writing to stdout
	"os"

	"github.com/ethereum-optimism/optimism/devnet-sdk/kt"
	"github.com/ethereum-optimism/optimism/devnet-sdk/manifest"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func main() {
	app := &cli.App{
		Name:  "devnet",
		Usage: "Generate Kurtosis parameters from a devnet manifest",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "manifest",
				Aliases:   []string{"m"},
				Usage:     "Path to the manifest YAML file",
				Required:  true,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Path to write the Kurtosis parameters file (default: stdout)",
			},
		},
		Action: func(c *cli.Context) error {
			manifestPath := c.String("manifest")
			
			// 1. Read manifest file
			manifestBytes, err := os.ReadFile(manifestPath)
			if err != nil {
				return fmt.Errorf("failed to read manifest file at %s: %w", manifestPath, err)
			}

			// 2. Parse manifest YAML
			var m manifest.Manifest
			if err := yaml.Unmarshal(manifestBytes, &m); err != nil {
				return fmt.Errorf("failed to parse manifest YAML from %s: %w", manifestPath, err)
			}

			// 3. Create visitor and process manifest
			visitor := kt.NewKurtosisVisitor()
			m.Accept(visitor)

			// 4. Get params and serialize to YAML
			params := visitor.GetParams()
			paramsBytes, err := yaml.Marshal(params)
			if err != nil {
				return fmt.Errorf("failed to marshal Kurtosis parameters to YAML: %w", err)
			}

			// 5. Write to file or stdout
			outputPath := c.String("output")
			if outputPath != "" {
				// Write to the specified file path
				// Use 0644 permissions (owner read/write, group/other read)
				if err := os.WriteFile(outputPath, paramsBytes, 0644); err != nil {
					return fmt.Errorf("failed to write parameters to output file %s: %w", outputPath, err)
				}
			} else {
				// Write directly to stdout using efficient io.Writer (os.Stdout)
				// Ensures the output is clean and efficient for piping.
				if _, err := io.WriteString(os.Stdout, string(paramsBytes)); err != nil {
					return fmt.Errorf("failed to write parameters to stdout: %w", err)
				}
			}

			return nil
		},
	}

	// Run the application and handle any top-level errors
	if err := app.Run(os.Args); err != nil {
		// Print error to Stderr and exit with status 1
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

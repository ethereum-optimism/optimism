package compare

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/urfave/cli/v2"
)

func CompareCLI() func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		comparator, err := NewComparatorFromCli(cliCtx)
		if err != nil {
			return err
		}

		addressDiffs, proofDiffs, err := comparator.Compare(cliCtx.Context)
		if err != nil {
			return fmt.Errorf("failed to validate: %w", err)
		}

		outputDir := cliCtx.String(OutputDirFlag.Name)
		if err := writeComparisonToJSON(addressDiffs, proofDiffs, outputDir); err != nil {
			return err
		}

		comparator.lgr.Info("completed comparing chain info", "outputDir", outputDir)
		return nil
	}
}

func (c *Comparator) Compare(ctx context.Context) (map[uint64]AddressDiffs, map[uint64]script.FaultProofStatus, error) {
	addressDiffs, err := c.CompareAddresses()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compare addresses: %w", err)
	}

	proofDiffs, err := c.CompareProofs()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compare proofs: %w", err)
	}

	return addressDiffs, proofDiffs, nil
}

// writeComparisonToJSON writes the comparison results to json files
func writeComparisonToJSON(addressDiffs map[uint64]AddressDiffs, proofDiffs map[uint64]script.FaultProofStatus, outputPath string) error {
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	addressData, err := json.MarshalIndent(addressDiffs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal comparison results: %w", err)
	}

	addressPath := filepath.Join(outputPath, "diff_addresses.json")
	if err := os.WriteFile(addressPath, addressData, 0644); err != nil {
		return fmt.Errorf("failed to write comparison results to file: %w", err)
	}

	proofData, err := json.MarshalIndent(proofDiffs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal comparison results: %w", err)
	}

	proofPath := filepath.Join(outputPath, "diff_proofs.json")
	if err := os.WriteFile(proofPath, proofData, 0644); err != nil {
		return fmt.Errorf("failed to write comparison results to file: %w", err)
	}

	return nil
}

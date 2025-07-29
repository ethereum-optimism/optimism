// Package main reproduces a lightweight version of the "kurtosis enclave inspect" command
// It can be used to sanity check the results, as writing tests against a fake
// enclave is not practical right now.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/inspect"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/util"
)

func main() {
	ctx := context.Background()

	var fixTraefik bool
	flag.BoolVar(&fixTraefik, "fix-traefik", false, "Fix missing Traefik labels on containers")

	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [--fix-traefik] <enclave-id>\n", os.Args[0])
		os.Exit(1)
	}

	enclaveID := flag.Arg(0)

	// If fix-traefik flag is provided, run the fix
	if fixTraefik {
		fmt.Println("🔧 Fixing Traefik network configuration...")
		if err := util.SetReverseProxyConfig(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error fixing Traefik network: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Traefik network configuration fixed!")
		return
	}

	inspector := inspect.NewInspector(enclaveID)

	data, err := inspector.ExtractData(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error inspecting enclave: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("File Artifacts:")
	for _, artifact := range data.FileArtifacts {
		fmt.Printf("  %s\n", artifact)
	}

	fmt.Println("\nServices:")
	for name, svc := range data.UserServices {
		fmt.Printf("  %s:\n", name)
		for portName, portInfo := range svc.Ports {
			host := portInfo.Host
			if host == "" {
				host = "localhost"
			}
			fmt.Printf("    %s: %s:%d\n", portName, host, portInfo.Port)
		}
	}
}

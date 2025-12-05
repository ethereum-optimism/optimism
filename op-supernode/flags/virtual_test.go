package flags

import (
	"flag"
	"fmt"
	"reflect"
	"testing"

	"github.com/urfave/cli/v2"

	opnodeflags "github.com/ethereum-optimism/optimism/op-node/flags"
)

func newTestContext(t *testing.T, flags []cli.Flag, args []string) *cli.Context {
	t.Helper()
	app := &cli.App{Flags: flags}
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, f := range flags {
		if err := f.Apply(set); err != nil {
			t.Fatalf("failed to apply flag %q: %v", f.Names()[0], err)
		}
	}
	if err := set.Parse(args); err != nil {
		t.Fatalf("failed to parse args %v: %v", args, err)
	}
	return cli.NewContext(app, set, nil)
}

func TestParseChainsVariants(t *testing.T) {
	tcs := []struct {
		name    string
		args    []string
		want    []uint64
		wantErr bool
	}{
		{
			name: "equals csv",
			args: []string{"--chains=1,2,3"},
			want: []uint64{1, 2, 3},
		},
		{
			name: "space csv with whitespace",
			args: []string{"--chains", "1, 2,3"},
			want: []uint64{1, 2, 3},
		},
		{
			name: "short equals",
			args: []string{"-chains=4"},
			want: []uint64{4},
		},
		{
			name: "repeated flags accumulate",
			args: []string{"--chains=5,6", "--chains", "7"},
			want: []uint64{5, 6, 7},
		},
		{
			name: "empty entries ignored",
			args: []string{"--chains=8,, 9, ,10"},
			want: []uint64{8, 9, 10},
		},
		{
			name:    "invalid value errors",
			args:    []string{"--chains=abc"},
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseChains(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (chains=%v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("chains mismatch: got %v want %v", got, tc.want)
			}
		})
	}
}

func TestFullDynamicFlags_ClonesAllFlagsForChainsAndGlobal(t *testing.T) {
	chains := []uint64{100, 200}
	flagsOut := FullDynamicFlags(chains)
	// collect primary names
	seen := make(map[string]struct{})
	for _, f := range flagsOut {
		name := f.Names()[0]
		seen[name] = struct{}{}
	}

	// For every op-node flag, we expect vn.all.<base> and vn.<id>.<base>
	for _, f := range opnodeflags.Flags {
		base := f.Names()[0]
		globalName := VNFlagGlobalPrefix + base
		if _, ok := seen[globalName]; !ok {
			t.Fatalf("missing global clone for %q: expected flag %q", base, globalName)
		}
		for _, id := range chains {
			perName := fmt.Sprintf("%s%d.%s", VNFlagNamePrefix, id, base)
			if _, ok := seen[perName]; !ok {
				t.Fatalf("missing per-chain clone for %q: expected flag %q", base, perName)
			}
		}
	}
}

func TestVirtualCLI_Precedence_String(t *testing.T) {
	flags := []cli.Flag{
		&cli.StringFlag{Name: VNFlagGlobalPrefix + "color", Value: "blue"},
		&cli.StringFlag{Name: "vn.1.color"}, // no default
	}

	// only defaults
	ctx := newTestContext(t, flags, []string{})
	v := NewVirtualCLI(ctx, 1)
	if got := v.String("color"); got != "blue" {
		t.Fatalf("default precedence failed: got %q want %q", got, "blue")
	}

	// set global value -> overrides default
	ctx = newTestContext(t, flags, []string{"--" + VNFlagGlobalPrefix + "color=green"})
	v = NewVirtualCLI(ctx, 1)
	if got := v.String("color"); got != "green" {
		t.Fatalf("global precedence failed: got %q want %q", got, "green")
	}

	// set chain-specific -> overrides global
	ctx = newTestContext(t, flags, []string{
		"--" + VNFlagGlobalPrefix + "color=green",
		"--vn.1.color=red",
	})
	v = NewVirtualCLI(ctx, 1)
	if got := v.String("color"); got != "red" {
		t.Fatalf("chain precedence failed: got %q want %q", got, "red")
	}

	// override -> overrides chain-specific
	v = v.WithStringOverride("color", "orange")
	if got := v.String("color"); got != "orange" {
		t.Fatalf("override precedence failed: got %q want %q", got, "orange")
	}
}

func TestVirtualCLI_Precedence_Bool(t *testing.T) {
	flags := []cli.Flag{
		&cli.BoolFlag{Name: VNFlagGlobalPrefix + "switch", Value: true},
		&cli.BoolFlag{Name: "vn.1.switch"}, // no default
	}

	// only defaults
	ctx := newTestContext(t, flags, []string{})
	v := NewVirtualCLI(ctx, 1)
	if got := v.Bool("switch"); got != true {
		t.Fatalf("default bool precedence failed: got %v want %v", got, true)
	}

	// set global value -> overrides default
	ctx = newTestContext(t, flags, []string{"--" + VNFlagGlobalPrefix + "switch=false"})
	v = NewVirtualCLI(ctx, 1)
	if got := v.Bool("switch"); got != false {
		t.Fatalf("global bool precedence failed: got %v want %v", got, false)
	}

	// set chain-specific -> overrides global
	ctx = newTestContext(t, flags, []string{
		"--" + VNFlagGlobalPrefix + "switch=false",
		"--vn.1.switch=true",
	})
	v = NewVirtualCLI(ctx, 1)
	if got := v.Bool("switch"); got != true {
		t.Fatalf("chain bool precedence failed: got %v want %v", got, true)
	}

	// override -> overrides chain-specific
	v = v.WithBoolOverride("switch", false)
	if got := v.Bool("switch"); got != false {
		t.Fatalf("override bool precedence failed: got %v want %v", got, false)
	}
}

func TestVirtualCLI_PerChainIsolation(t *testing.T) {
	flags := []cli.Flag{
		&cli.StringFlag{Name: VNFlagGlobalPrefix + "color", Value: "blue"},
		&cli.StringFlag{Name: "vn.1.color"},
		&cli.StringFlag{Name: "vn.2.color"},
	}
	ctx := newTestContext(t, flags, []string{"--vn.1.color=red", "--vn.2.color=green"})
	v1 := NewVirtualCLI(ctx, 1)
	v2 := NewVirtualCLI(ctx, 2)
	if got := v1.String("color"); got != "red" {
		t.Fatalf("chain 1 value mismatch: got %q want %q", got, "red")
	}
	if got := v2.String("color"); got != "green" {
		t.Fatalf("chain 2 value mismatch: got %q want %q", got, "green")
	}
}

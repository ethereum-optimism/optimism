package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/oprm/flags"
	"github.com/ethereum-optimism/optimism/oprm/manager"
	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/ethereum-optimism/optimism/oprm/tui"
	"github.com/ethereum-optimism/optimism/oprm/workflow"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

var (
	Version   = ""
	GitCommit = ""
	GitDate   = ""
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	if err := run(ctx, os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	app := cli.NewApp()
	app.Name = "oprm"
	app.Usage = "Optimism release manager"
	app.Description = "Stateful release workflow manager for OP Stack component releases."
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Writer = stdout
	app.ErrWriter = stderr
	app.Flags = flags.GlobalFlags()
	app.Action = func(c *cli.Context) error {
		return cli.ShowAppHelp(c)
	}
	app.Commands = []*cli.Command{
		doctorCommand(),
		discoverCommand(),
		runCommand(),
		planCommand(),
		resumeCommand(),
		statusCommand(),
		retryCommand(),
		skipCommand(),
		satisfyCommand(),
		logCommand(),
	}
	return app.RunContext(ctx, args)
}

func newManagerFromCLI(c *cli.Context) (*manager.App, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve current working directory: %w", err)
	}
	if err := manager.ValidateMonorepoRoot(wd); err != nil {
		return nil, err
	}
	cfg, err := manager.LoadConfig(c.Path(flags.ConfigFlag.Name), manager.ConfigOverrides{
		RunsDir:            c.Path(flags.RunsDirFlag.Name),
		BaseBranch:         c.String(flags.BaseBranchFlag.Name),
		GitHubOwner:        c.String(flags.GitHubOwnerFlag.Name),
		GitHubRepo:         c.String(flags.GitHubRepoFlag.Name),
		OpGethOwner:        c.String(flags.OpGethOwnerFlag.Name),
		OpGethRepo:         c.String(flags.OpGethRepoFlag.Name),
		OpGethCheckoutPath: c.Path(flags.OpGethCheckoutFlag.Name),
	})
	if err != nil {
		return nil, err
	}
	return manager.New(cfg, c.App.Writer, c.App.ErrWriter), nil
}

func doctorCommand() *cli.Command {
	return &cli.Command{
		Name:  "doctor",
		Usage: "Check prerequisite tooling and detect the release manager",
		Action: func(c *cli.Context) error {
			app, err := newManagerFromCLI(c)
			if err != nil {
				return err
			}
			report, err := app.Doctor(c.Context)
			if err != nil {
				return err
			}
			for _, check := range report.Checks {
				icon := "OK"
				if check.Status != workflow.StatusCompleted {
					icon = "FAIL"
				}
				fmt.Fprintf(c.App.Writer, "[%s] %s: %s\n", icon, check.Title, check.Detail)
			}
			fmt.Fprintf(c.App.Writer, "release manager: %s\n", report.ReleaseManager.String())
			if report.Blocking() {
				return fmt.Errorf("doctor found blocking issues")
			}
			return nil
		},
	}
}

func discoverCommand() *cli.Command {
	return &cli.Command{
		Name:      "discover",
		Usage:     "Discover the latest release and RC versions for components",
		ArgsUsage: "[component-id ...]",
		Action: func(c *cli.Context) error {
			app, err := newManagerFromCLI(c)
			if err != nil {
				return err
			}
			items, err := app.DiscoverComponentVersions(c.Context, c.Args().Slice())
			if err != nil {
				return err
			}
			for _, item := range items {
				latestRelease := "<none>"
				if item.LatestRelease != nil {
					latestRelease = formatMatchedRelease(item.LatestRelease)
				}
				latestRC := "<none>"
				if item.LatestRC != nil {
					latestRC = formatMatchedRelease(item.LatestRC)
				}
				fmt.Fprintf(c.App.Writer, "%s\n", item.Component.ID)
				fmt.Fprintf(c.App.Writer, "  repo: %s/%s\n", item.Component.GitHubOwner, item.Component.GitHubRepo)
				fmt.Fprintf(c.App.Writer, "  latest release: %s\n", latestRelease)
				fmt.Fprintf(c.App.Writer, "  latest rc: %s\n", latestRC)
			}
			return nil
		},
	}
}

func formatMatchedRelease(item *release.MatchedRelease) string {
	if item == nil {
		return "<none>"
	}
	value := item.Version
	if item.Draft {
		value += " (draft)"
	} else if item.PreRelease {
		value += " (pre-release)"
	}
	return value
}

func planCommand() *cli.Command {
	return &cli.Command{
		Name:      "plan",
		Usage:     "Generate and persist a release plan for selected components",
		ArgsUsage: "<run-id-or-path> [component-id ...]",
		Flags:     []cli.Flag{flags.BumpFlag, flags.ManualVersionFlag},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("expected a run id/path and optional component ids")
			}
			app, err := newManagerFromCLI(c)
			if err != nil {
				return err
			}
			bumps, err := parseBumpAssignments(c.StringSlice(flags.BumpFlag.Name))
			if err != nil {
				return err
			}
			manualTargets, err := parseStringAssignments(c.StringSlice(flags.ManualVersionFlag.Name))
			if err != nil {
				return err
			}
			run, path, results, err := app.PlanRun(c.Context, c.Args().Get(0), c.Args().Slice()[1:], manager.PlanOptions{
				Bumps:         bumps,
				ManualTargets: manualTargets,
			})
			if err != nil {
				return err
			}
			printRunSummary(c.App.Writer, run, path)
			fmt.Fprintln(c.App.Writer, "planned components:")
			for _, item := range results {
				fmt.Fprintf(c.App.Writer, "- %s: changed=%t", item.ComponentID, item.Proposal.Changed)
				if item.Proposal.Proposed != "" {
					fmt.Fprintf(c.App.Writer, ", target=%s, rc=%s", item.Proposal.TargetRelease, item.Proposal.Proposed)
				}
				fmt.Fprintln(c.App.Writer)
			}
			return nil
		},
	}
}

func runCommand() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Initialize a new release run, auto-detect component changes, and open the terminal UI",
		Action: func(c *cli.Context) error {
			app, err := newManagerFromCLI(c)
			if err != nil {
				return err
			}
			run, path, report, err := app.CreateRun(c.Context)
			if err != nil {
				return err
			}
			if report.Blocking() {
				printRunSummary(c.App.Writer, run, path)
				fmt.Fprintln(c.App.Writer, "run is blocked until doctor issues are resolved")
				return nil
			}
			run, path, _, err = app.PlanRun(c.Context, run.RunID, nil, manager.PlanOptions{})
			if err != nil {
				return err
			}
			printRunSummary(c.App.Writer, run, path)
			if !canOpenTUI(c.App.Writer) {
				fmt.Fprintf(c.App.Writer, "terminal UI not opened because stdout is not a TTY; run 'oprm resume %s' in a terminal\n", run.RunID)
				return nil
			}
			fmt.Fprintln(c.App.Writer, "opening terminal UI for component selection")
			return tui.Run(app, run.RunID)
		},
	}
}

func resumeCommand() *cli.Command {
	return &cli.Command{
		Name:      "resume",
		Usage:     "Resume an existing release run in the terminal UI",
		ArgsUsage: "<run-id-or-path>",
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return fmt.Errorf("expected exactly one run id or path")
			}
			app, err := newManagerFromCLI(c)
			if err != nil {
				return err
			}
			return tui.Run(app, c.Args().First())
		},
	}
}

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "Show the status of an existing release run",
		ArgsUsage: "<run-id-or-path>",
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return fmt.Errorf("expected exactly one run id or path")
			}
			app, err := newManagerFromCLI(c)
			if err != nil {
				return err
			}
			run, path, err := app.LoadRun(c.Args().First())
			if err != nil {
				return err
			}
			printRunSummary(c.App.Writer, run, path)
			return nil
		},
	}
}

func retryCommand() *cli.Command {
	return &cli.Command{
		Name:      "retry",
		Usage:     "Reset a task to pending for retry",
		ArgsUsage: "<run-id-or-path> <task-id>",
		Action: func(c *cli.Context) error {
			if c.NArg() != 2 {
				return fmt.Errorf("expected a run id/path and a task id")
			}
			app, err := newManagerFromCLI(c)
			if err != nil {
				return err
			}
			run, path, err := app.RetryTask(c.Args().Get(0), c.Args().Get(1))
			if err != nil {
				return err
			}
			printRunSummary(c.App.Writer, run, path)
			return nil
		},
	}
}

func skipCommand() *cli.Command {
	return &cli.Command{
		Name:      "skip",
		Usage:     "Mark a task as skipped",
		ArgsUsage: "<run-id-or-path> <task-id> <reason>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 3 {
				return fmt.Errorf("expected a run id/path, a task id, and a reason")
			}
			app, err := newManagerFromCLI(c)
			if err != nil {
				return err
			}
			run, path, err := app.SkipTask(c.Args().Get(0), c.Args().Get(1), strings.Join(c.Args().Slice()[2:], " "))
			if err != nil {
				return err
			}
			printRunSummary(c.App.Writer, run, path)
			return nil
		},
	}
}

func satisfyCommand() *cli.Command {
	return &cli.Command{
		Name:      "satisfy",
		Usage:     "Mark a task as externally satisfied",
		ArgsUsage: "<run-id-or-path> <task-id> <reason>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 3 {
				return fmt.Errorf("expected a run id/path, a task id, and a reason")
			}
			app, err := newManagerFromCLI(c)
			if err != nil {
				return err
			}
			run, path, err := app.SatisfyTask(c.Args().Get(0), c.Args().Get(1), strings.Join(c.Args().Slice()[2:], " "))
			if err != nil {
				return err
			}
			printRunSummary(c.App.Writer, run, path)
			return nil
		},
	}
}

func logCommand() *cli.Command {
	return &cli.Command{
		Name:      "log",
		Usage:     "Print the journal path or contents for a release run",
		ArgsUsage: "<run-id-or-path>",
		Flags:     []cli.Flag{flags.PrintFlag},
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return fmt.Errorf("expected exactly one run id or path")
			}
			app, err := newManagerFromCLI(c)
			if err != nil {
				return err
			}
			_, path, err := app.LoadRun(c.Args().First())
			if err != nil {
				return err
			}
			if !c.Bool(flags.PrintFlag.Name) {
				fmt.Fprintln(c.App.Writer, path)
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			_, err = c.App.Writer.Write(data)
			return err
		},
	}
}

func printRunSummary(w io.Writer, run *release.Run, path string) {
	fmt.Fprintf(w, "run id: %s\n", run.RunID)
	fmt.Fprintf(w, "status: %s\n", run.Status)
	fmt.Fprintf(w, "journal: %s\n", path)
	fmt.Fprintf(w, "release manager: %s\n", run.ReleaseManager.String())
	fmt.Fprintf(w, "candidates: %d\n", len(run.Candidates))
	fmt.Fprintf(w, "selected components: %d (confirmed=%t)\n", len(run.Components), run.SelectionConfirmed)
	for _, componentID := range run.Components {
		proposal, ok := run.Versions[componentID]
		if !ok {
			fmt.Fprintf(w, "- %s\n", componentID)
			continue
		}
		fmt.Fprintf(w, "- %s (changed=%t", componentID, proposal.Changed)
		if proposal.Proposed != "" {
			fmt.Fprintf(w, ", target=%s, rc=%s", proposal.TargetRelease, proposal.Proposed)
		}
		if proposal.ResumeDraft {
			fmt.Fprint(w, ", resume-draft=true")
		}
		fmt.Fprintln(w, ")")
	}
	fmt.Fprintf(w, "tasks: %d\n", len(run.Tasks))
	for _, task := range run.Tasks {
		fmt.Fprintf(w, "- %s [%s]", task.ID, task.Status)
		if task.Reason != "" {
			fmt.Fprintf(w, " - %s", task.Reason)
		}
		fmt.Fprintln(w)
	}
}

func canOpenTUI(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func parseStringAssignments(values []string) (map[string]string, error) {
	out := make(map[string]string, len(values))
	for _, value := range values {
		key, parsed, ok := strings.Cut(value, "=")
		if !ok || key == "" || parsed == "" {
			return nil, fmt.Errorf("invalid assignment %q, expected component=value", value)
		}
		out[key] = parsed
	}
	return out, nil
}

func parseBumpAssignments(values []string) (map[string]release.BumpKind, error) {
	raw, err := parseStringAssignments(values)
	if err != nil {
		return nil, err
	}
	out := make(map[string]release.BumpKind, len(raw))
	for componentID, value := range raw {
		bump := release.BumpKind(value)
		switch bump {
		case release.BumpPatch, release.BumpMinor, release.BumpMajor:
			out[componentID] = bump
		default:
			return nil, fmt.Errorf("invalid bump %q for %s", value, componentID)
		}
	}
	return out, nil
}

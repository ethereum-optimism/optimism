package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/ethereum-optimism/optimism/oprm/workflow"
	"github.com/stretchr/testify/require"
)

func TestListWindow(t *testing.T) {
	start, end := listWindow(10, 8, 5)
	require.Equal(t, 5, start)
	require.Equal(t, 10, end)

	start, end = listWindow(10, 0, 5)
	require.Equal(t, 0, start)
	require.Equal(t, 5, end)
}

func TestRenderComponentListScrollsSelection(t *testing.T) {
	run := release.NewRun("run-1", time.Now(), "ethereum-optimism/optimism", "develop", release.ReleaseManager{}, ".oprm/releases")
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("component-%d", i)
		run.Candidates = append(run.Candidates, id)
		run.Versions[id] = release.VersionProposal{Changed: i%2 == 0}
	}
	run.Components = []string{"component-8"}

	m := &Model{run: run, selectedComp: 7, height: 14}
	view := m.renderComponentList()
	require.Contains(t, view, "Components to release (6-10/10)")
	require.Contains(t, view, "> [x] component-8 (changed)")
	require.NotContains(t, view, "\n  [ ] component-1 ")
}

func TestWrapCommandLinePreservesFullCommand(t *testing.T) {
	line := "git -C /tmp/optimism push upstream op-node/v1.2.4-rc.1"
	wrapped := wrapCommandLine(line, 20)
	require.Greater(t, len(wrapped), 1)
	require.Equal(t, line, strings.Join(wrapped, ""))
	for _, part := range wrapped {
		require.LessOrEqual(t, len(part), 22)
	}
}

func TestRenderTaskListScrollsSelection(t *testing.T) {
	run := release.NewRun("run-1", time.Now(), "ethereum-optimism/optimism", "develop", release.ReleaseManager{}, ".oprm/releases")
	for i := 1; i <= 10; i++ {
		run.Tasks = append(run.Tasks, workflow.TaskState{ID: fmt.Sprintf("task-%d", i), Status: workflow.StatusPending})
	}

	m := &Model{run: run, selectedTask: 8, height: 14}
	view := m.renderTaskList()
	require.Contains(t, view, "Tasks (6-10/10)")
	require.True(t, strings.Contains(view, "> ") && strings.Contains(view, "task-9"))
	require.NotContains(t, view, "task-1\n")
}

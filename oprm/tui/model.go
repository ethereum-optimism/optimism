package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	chansi "github.com/charmbracelet/x/ansi"
	"github.com/ethereum-optimism/optimism/oprm/manager"
	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/ethereum-optimism/optimism/oprm/workflow"
)

type mode int

const (
	modeNormal mode = iota
	modePrompt
	modeConfirm
)

type Model struct {
	app          *manager.App
	identifier   string
	run          *release.Run
	path         string
	selectedTask int
	selectedComp int
	width        int
	height       int
	mode         mode
	promptVerb   string
	promptTask   string
	promptText   string
	statusMsg    string
	errMsg       string
}

func New(app *manager.App, identifier string) (*Model, error) {
	run, path, err := app.LoadRun(identifier)
	if err != nil {
		return nil, err
	}
	return &Model{app: app, identifier: identifier, run: run, path: path}, nil
}

func Run(app *manager.App, identifier string) error {
	model, err := New(app, identifier)
	if err != nil {
		return err
	}
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modePrompt:
			return m.handlePromptKey(msg)
		case modeConfirm:
			return m.handleConfirmKey(msg)
		default:
			if m.inSelectionMode() {
				return m.handleSelectionKey(msg)
			}
			return m.handleTaskKey(msg)
		}
	}
	return m, nil
}

func (m *Model) inSelectionMode() bool {
	return m.run != nil && !m.run.SelectionConfirmed
}

func (m *Model) handleSelectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.selectedComp > 0 {
			m.selectedComp--
		}
	case "down", "j":
		if m.selectedComp < len(m.run.Candidates)-1 {
			m.selectedComp++
		}
	case " ":
		if id := m.currentComponentID(); id != "" {
			m.toggleComponent(id)
		}
	case "a":
		m.selectAllChanged()
	case "g":
		m.reload()
	case "enter":
		_, _, err := m.app.UpdateSelection(m.identifier, m.run.Components, true)
		m.finishMutation(err, "component selection confirmed")
	}
	return m, nil
}

func (m *Model) handleTaskKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.selectedTask > 0 {
			m.selectedTask--
		}
	case "down", "j":
		if m.selectedTask < len(m.run.Tasks)-1 {
			m.selectedTask++
		}
	case "g":
		m.reload()
	case "r":
		task := m.currentTask()
		if task != nil {
			_, _, err := m.app.RetryTask(m.identifier, task.ID)
			m.finishMutation(err, fmt.Sprintf("reset %s for retry", task.ID))
		}
	case "s":
		if task := m.currentTask(); task != nil {
			m.mode = modePrompt
			m.promptVerb = "skip"
			m.promptTask = task.ID
			m.promptText = ""
		}
	case "e":
		if task := m.currentTask(); task != nil {
			m.mode = modePrompt
			m.promptVerb = "mark externally satisfied"
			m.promptTask = task.ID
			m.promptText = ""
		}
	case "enter":
		task := m.currentTask()
		if task == nil {
			return m, nil
		}
		if requiresConfirmation(task.ID) {
			m.mode = modeConfirm
			return m, nil
		}
		err := m.app.ExecuteTask(m.identifier, task.ID)
		m.finishMutation(err, fmt.Sprintf("executed %s", task.ID))
	}
	return m, nil
}

func (m *Model) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.promptText = ""
		m.statusMsg = "cancelled"
		m.errMsg = ""
	case "enter":
		reason := strings.TrimSpace(m.promptText)
		if reason == "" {
			m.errMsg = "reason is required"
			return m, nil
		}
		var err error
		switch m.promptVerb {
		case "skip":
			_, _, err = m.app.SkipTask(m.identifier, m.promptTask, reason)
		default:
			_, _, err = m.app.SatisfyTask(m.identifier, m.promptTask, reason)
		}
		m.mode = modeNormal
		m.promptText = ""
		m.finishMutation(err, fmt.Sprintf("%s %s", m.promptVerb, m.promptTask))
	case "backspace":
		if len(m.promptText) > 0 {
			m.promptText = m.promptText[:len(m.promptText)-1]
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.promptText += msg.String()
		}
	}
	return m, nil
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.mode = modeNormal
		m.statusMsg = "confirmation cancelled"
		m.errMsg = ""
	case "enter", "y":
		task := m.currentTask()
		m.mode = modeNormal
		if task != nil {
			err := m.app.ExecuteTask(m.identifier, task.ID)
			m.finishMutation(err, fmt.Sprintf("confirmed %s", task.ID))
		}
	}
	return m, nil
}

func (m *Model) finishMutation(err error, status string) {
	if err != nil {
		m.errMsg = err.Error()
		m.statusMsg = ""
		return
	}
	m.statusMsg = status
	m.errMsg = ""
	m.reload()
}

func (m *Model) reload() {
	run, path, err := m.app.LoadRun(m.identifier)
	if err != nil {
		m.errMsg = err.Error()
		return
	}
	m.run = run
	m.path = path
	if m.selectedTask >= len(m.run.Tasks) && len(m.run.Tasks) > 0 {
		m.selectedTask = len(m.run.Tasks) - 1
	}
	if m.selectedComp >= len(m.run.Candidates) && len(m.run.Candidates) > 0 {
		m.selectedComp = len(m.run.Candidates) - 1
	}
}

func (m *Model) currentTask() *workflow.TaskState {
	if len(m.run.Tasks) == 0 || m.selectedTask < 0 || m.selectedTask >= len(m.run.Tasks) {
		return nil
	}
	return &m.run.Tasks[m.selectedTask]
}

func (m *Model) currentComponentID() string {
	if len(m.run.Candidates) == 0 || m.selectedComp < 0 || m.selectedComp >= len(m.run.Candidates) {
		return ""
	}
	return m.run.Candidates[m.selectedComp]
}

func (m *Model) toggleComponent(id string) {
	selected := make(map[string]struct{}, len(m.run.Components))
	for _, item := range m.run.Components {
		selected[item] = struct{}{}
	}
	if _, ok := selected[id]; ok {
		delete(selected, id)
	} else {
		selected[id] = struct{}{}
	}
	m.run.Components = m.orderSelected(selected)
}

func (m *Model) selectAllChanged() {
	selected := make(map[string]struct{})
	for _, id := range m.run.Candidates {
		if proposal, ok := m.run.Versions[id]; ok && proposal.Changed {
			selected[id] = struct{}{}
		}
	}
	m.run.Components = m.orderSelected(selected)
}

func (m *Model) orderSelected(selected map[string]struct{}) []string {
	ordered := make([]string, 0, len(selected))
	for _, id := range m.run.Candidates {
		if _, ok := selected[id]; ok {
			ordered = append(ordered, id)
		}
	}
	return ordered
}

func (m *Model) View() string {
	if m.run == nil {
		return "loading..."
	}
	width := m.width
	if width == 0 {
		width = 140
	}
	height := m.height
	if height == 0 {
		height = 40
	}
	headline := fmt.Sprintf("OPRM  run=%s  manager=%s", m.run.RunID, m.run.ReleaseManager.String())
	if m.inSelectionMode() {
		headline += "  stage=component-selection"
	} else {
		headline += "  stage=tasks"
	}
	targets := fmt.Sprintf("targets  monorepo=%s/%s@%s  op-geth=%s/%s@%s",
		emptyFallback(m.app.Config.GitHub.Owner),
		emptyFallback(m.app.Config.GitHub.Repo),
		emptyFallback(m.app.Config.BaseBranch),
		emptyFallback(m.app.Config.OpGeth.Owner),
		emptyFallback(m.app.Config.OpGeth.Repo),
		emptyFallback(opGethBaseBranch(m.app)),
	)
	header := headerStyle.Render(headline + "  journal=" + m.path + "\n" + targets)
	leftWidth := width / 3
	if leftWidth < 38 {
		leftWidth = 38
	}
	rightWidth := width - leftWidth - 3
	left := panelStyle.Width(leftWidth).Height(height - 6).Render(fitPaneContent(m.renderLeftPane(), leftWidth-4))
	right := panelStyle.Width(rightWidth).Height(height - 6).Render(fitPaneContent(m.renderDetails(rightWidth-4), rightWidth-4))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	footer := footerStyle.Render(m.renderFooter())
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *Model) renderLeftPane() string {
	if m.inSelectionMode() {
		return m.renderComponentList()
	}
	return m.renderTaskList()
}

func (m *Model) renderComponentList() string {
	selectedSet := make(map[string]struct{}, len(m.run.Components))
	for _, id := range m.run.Components {
		selectedSet[id] = struct{}{}
	}
	visibleItems := m.leftPaneVisibleRows() - 1
	start, end := listWindow(len(m.run.Candidates), m.selectedComp, visibleItems)
	title := "Components to release"
	if len(m.run.Candidates) > 0 && end-start < len(m.run.Candidates) {
		title = fmt.Sprintf("Components to release (%d-%d/%d)", start+1, end, len(m.run.Candidates))
	}
	lines := []string{titleStyle.Render(title)}
	for i := start; i < end; i++ {
		id := m.run.Candidates[i]
		cursor := " "
		if i == m.selectedComp {
			cursor = ">"
		}
		mark := "[ ]"
		if _, ok := selectedSet[id]; ok {
			mark = "[x]"
		}
		proposal := m.run.Versions[id]
		changed := "unchanged"
		if proposal.Changed {
			changed = "changed"
		}
		lines = append(lines, fmt.Sprintf("%s %s %s (%s)", cursor, mark, id, changed))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderTaskList() string {
	visibleItems := m.leftPaneVisibleRows() - 1
	start, end := listWindow(len(m.run.Tasks), m.selectedTask, visibleItems)
	title := "Tasks"
	if len(m.run.Tasks) > 0 && end-start < len(m.run.Tasks) {
		title = fmt.Sprintf("Tasks (%d-%d/%d)", start+1, end, len(m.run.Tasks))
	}
	lines := []string{titleStyle.Render(title)}
	for i := start; i < end; i++ {
		task := m.run.Tasks[i]
		cursor := " "
		if i == m.selectedTask {
			cursor = ">"
		}
		lines = append(lines, fmt.Sprintf("%s %s %s", cursor, paddedStatusStyle(task.Status, 22), task.ID))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderDetails(contentWidth int) string {
	if m.inSelectionMode() {
		return m.renderComponentDetails(m.currentComponentID())
	}
	task := m.currentTask()
	if task == nil {
		return titleStyle.Render("No tasks")
	}
	componentID := strings.SplitN(task.ID, ".", 2)[0]
	lines := []string{titleStyle.Render(task.ID)}
	if task.Title != "" {
		lines = append(lines, fmt.Sprintf("Title: %s", task.Title))
	}
	lines = append(lines, fmt.Sprintf("Status: %s", task.Status))
	if desc := m.taskDescription(task.ID); desc != "" {
		lines = append(lines, fmt.Sprintf("Purpose: %s", desc))
	}
	if task.Reason != "" {
		lines = append(lines, fmt.Sprintf("Reason: %s", task.Reason))
	}
	lines = append(lines, "", m.renderComponentDetails(componentID))
	if commands, err := m.app.PreviewTaskCommands(m.run, task.ID); err != nil {
		lines = append(lines, "", subtitleStyle.Render("Command preview"), errorStyle.Render(err.Error()))
	} else if len(commands) > 0 {
		lines = append(lines, "", subtitleStyle.Render("Command preview"))
		lines = append(lines, renderCommandPreview(commands, contentWidth)...)
	}
	if info := m.taskContextDetails(task.ID); len(info) > 0 {
		lines = append(lines, "", subtitleStyle.Render("Task context"))
		lines = append(lines, info...)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderComponentDetails(componentID string) string {
	if componentID == "" {
		return titleStyle.Render("No component selected")
	}
	proposal := m.run.Versions[componentID]
	lines := []string{
		titleStyle.Render(componentID),
		fmt.Sprintf("Latest release: %s", emptyFallback(proposal.LatestRelease)),
		fmt.Sprintf("Latest RC: %s", emptyFallback(proposal.LatestRC)),
		fmt.Sprintf("Latest draft RC: %s", emptyFallback(proposal.LatestDraftRC)),
		fmt.Sprintf("Changed: %t", proposal.Changed),
	}
	if proposal.ResumeDraft {
		lines = append(lines, "Resuming existing draft RC: yes")
	}
	if proposal.TargetRelease != "" {
		lines = append(lines, fmt.Sprintf("Target release: %s", remoteTagStyled(proposal.TargetRelease, proposal.TargetTagRemoteState)))
	}
	if proposal.Proposed != "" {
		lines = append(lines, fmt.Sprintf("Proposed RC: %s", remoteTagStyled(proposal.Proposed, proposal.ProposedTagRemoteState)))
	}
	lines = append(lines,
		"",
		subtitleStyle.Render("Review range"),
		fmt.Sprintf("From: %s", emptyFallback(proposal.Review.FromRef)),
		fmt.Sprintf("To: %s (%s)", emptyFallback(proposal.Review.ToRef), emptyFallback(proposal.Review.ToRefKind)),
	)
	if proposal.Review.CompareURL != "" {
		lines = append(lines, fmt.Sprintf("Compare: %s", proposal.Review.CompareURL))
	}
	lines = append(lines, "", subtitleStyle.Render("Commits"))
	for _, item := range proposal.Review.CommitSummaries {
		lines = append(lines, "- "+item)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderFooter() string {
	if m.mode == modePrompt {
		return fmt.Sprintf("%s %s: %s", m.promptVerb, m.promptTask, m.promptText)
	}
	if m.mode == modeConfirm {
		return "Confirm selected task and run the previewed command(s)? enter/y=yes esc/n=no"
	}
	status := m.statusMsg
	if m.errMsg != "" {
		status = errorStyle.Render(m.errMsg)
	}
	if status != "" {
		return status
	}
	if m.inSelectionMode() {
		return "j/k=move  space=toggle component  a=select all changed  enter=confirm selection  g=refresh  q=quit"
	}
	return "j/k=move  enter=execute  r=retry  s=skip  e=external  g=refresh  q=quit"
}

func emptyFallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<none>"
	}
	return value
}

func requiresConfirmation(taskID string) bool {
	switch {
	case strings.HasSuffix(taskID, ".review-diff"):
		return true
	case strings.HasSuffix(taskID, ".create-tag"):
		return true
	case strings.HasSuffix(taskID, ".push-tag"):
		return true
	case strings.HasSuffix(taskID, ".github-draft-release"):
		return true
	case strings.HasSuffix(taskID, ".manual-confirm-builds-ready"):
		return true
	case strings.HasSuffix(taskID, ".rollout"):
		return true
	case strings.HasSuffix(taskID, ".finalize-release"):
		return true
	default:
		return false
	}
}

func (m *Model) taskDescription(taskID string) string {
	switch taskID {
	case "doctor.gh-installed":
		return "Required for GitHub API access and later release operations."
	case "doctor.gh-authenticated":
		return "Confirms gh is logged in as the acting release manager for this run."
	case "doctor.git-installed":
		return "Required for local checkout checks and later tag-related git actions."
	case "doctor.git-configured":
		return "Ensures git user.name and user.email are available for auditability and future mutations."
	case "doctor.git-fetch-tags-monorepo":
		return "Fetches all tags from the local git remote that matches the configured monorepo target so local tag state is current before planning and tag operations."
	case "doctor.git-fetch-tags-op-geth":
		return "Fetches all tags from the local git remote that matches the configured op-geth target so external tag operations see current tag state."
	case "doctor.monorepo-base-branch-synced":
		return "Confirms the local monorepo checkout is on the configured base branch and that HEAD is a releasable commit for the local remote matching the configured target: either matching that remote or an older ancestor suitable for resuming an in-progress release."
	case "doctor.op-geth-base-branch-synced":
		return "Confirms the local op-geth checkout is on its base branch and that HEAD is a releasable commit for the local remote matching the configured target: either matching that remote or an older ancestor suitable for resuming an in-progress release."
	case "doctor.release-manager-detected":
		return "Binds the run to the current GitHub and git identity."
	case "doctor.repo-push-monorepo":
		return "Confirms the acting user can push to the configured monorepo target."
	case "doctor.repo-push-op-geth":
		return "Confirms the acting user can push to the configured op-geth target."
	}

	componentID, suffix, ok := strings.Cut(taskID, ".")
	if !ok {
		return ""
	}
	proposal := m.run.Versions[componentID]

	switch suffix {
	case "review-diff":
		return fmt.Sprintf("Review release scope for %s: in-scope changes, target release %s, and proposed RC %s.", componentID, emptyFallback(proposal.TargetRelease), emptyFallback(proposal.Proposed))
	case "prepare-release-notes":
		return "Write a draft release-notes artifact under the run directory for operator review and later release publishing."
	case "create-tag":
		return fmt.Sprintf("Create the proposed RC tag %s locally only. This does not push anything to the remote repository.", emptyFallback(proposal.Proposed))
	case "push-tag":
		return fmt.Sprintf("Push the already-created local RC tag %s to the configured remote repository and verify it exists remotely.", emptyFallback(proposal.Proposed))
	case "github-draft-release":
		return fmt.Sprintf("Create or update the GitHub draft release for %s using the generated release-notes artifact.", emptyFallback(proposal.Proposed))
	case "manual-confirm-builds-ready":
		return fmt.Sprintf("Human checkpoint to confirm builds, images, and jobs are ready after %s has been tagged and the draft release exists.", emptyFallback(proposal.Proposed))
	case "rollout":
		return fmt.Sprintf("Manual checkpoint to trigger ./op rollout for %s before finalizing the release.", emptyFallback(proposal.TargetRelease))
	case "finalize-release":
		return fmt.Sprintf("Create and push the final release tag %s at the validated RC %s commit, then retarget and publish the GitHub release.", emptyFallback(proposal.TargetRelease), emptyFallback(proposal.Proposed))
	default:
		return ""
	}
}

func (m *Model) taskContextDetails(taskID string) []string {
	componentID, suffix, ok := strings.Cut(taskID, ".")
	if !ok || strings.HasPrefix(taskID, "doctor.") {
		return nil
	}
	spec, err := m.app.ConfiguredComponentSpec(componentID)
	if err != nil {
		return nil
	}
	proposal := m.run.Versions[componentID]
	lines := []string{
		fmt.Sprintf("- Repo target: %s/%s", emptyFallback(spec.GitHubOwner), emptyFallback(spec.GitHubRepo)),
		fmt.Sprintf("- Branch target: %s", emptyFallback(spec.BaseBranch)),
		fmt.Sprintf("- Local checkout: %s", emptyFallback(m.app.CheckoutLocation(componentID))),
	}
	switch suffix {
	case "review-diff", "create-tag", "push-tag", "github-draft-release", "manual-confirm-builds-ready", "rollout", "finalize-release":
		lines = append(lines,
			fmt.Sprintf("- Target release: %s", remoteTagStyled(emptyFallback(proposal.TargetRelease), proposal.TargetTagRemoteState)),
			fmt.Sprintf("- Proposed RC: %s", remoteTagStyled(emptyFallback(proposal.Proposed), proposal.ProposedTagRemoteState)),
		)
		if suffix == "github-draft-release" || suffix == "finalize-release" {
			lines = append(lines, fmt.Sprintf("- Release notes artifact: %s", emptyFallback(m.app.ReleaseNotesPath(m.run, componentID))))
		}
	case "prepare-release-notes":
		lines = append(lines,
			fmt.Sprintf("- Review range: %s -> %s", emptyFallback(proposal.Review.FromRef), emptyFallback(proposal.Review.ToRef)),
			fmt.Sprintf("- Output artifact: %s", emptyFallback(m.app.ReleaseNotesPath(m.run, componentID))),
		)
	}
	return lines
}

func remoteTagStyled(value string, state string) string {
	switch state {
	case "missing":
		return missingTagStyle.Render(value)
	case "exists":
		return existingTagStyle.Render(value)
	default:
		return value
	}
}

func opGethBaseBranch(app *manager.App) string {
	spec, err := app.ConfiguredComponentSpec("op-geth")
	if err != nil {
		return ""
	}
	return spec.BaseBranch
}

func (m *Model) leftPaneVisibleRows() int {
	height := m.height
	if height == 0 {
		height = 40
	}
	rows := height - 6 - panelStyle.GetVerticalFrameSize()
	if rows < 2 {
		return 2
	}
	return rows
}

func listWindow(total int, selected int, visible int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if visible <= 0 || total <= visible {
		return 0, total
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= total {
		selected = total - 1
	}
	start := selected - visible/2
	if start < 0 {
		start = 0
	}
	if start+visible > total {
		start = total - visible
	}
	return start, start + visible
}

func renderCommandPreview(commands []string, width int) []string {
	if len(commands) == 0 {
		return nil
	}
	codeWidth := width - commandStyle.GetHorizontalFrameSize() - 2
	if codeWidth < 12 {
		codeWidth = 12
	}
	var lines []string
	for _, command := range commands {
		wrapped := wrapCommandLine(command, codeWidth)
		for i, line := range wrapped {
			if i > 0 {
				line = "  " + line
			}
			lines = append(lines, commandStyle.Render(line))
		}
	}
	return lines
}

func wrapCommandLine(line string, width int) []string {
	if width <= 0 || chansi.StringWidth(line) <= width {
		return []string{line}
	}
	var out []string
	remaining := line
	for chansi.StringWidth(remaining) > width {
		splitAt := bestWrapIndex(remaining, width)
		if splitAt <= 0 || splitAt >= len(remaining) {
			break
		}
		out = append(out, remaining[:splitAt])
		remaining = remaining[splitAt:]
	}
	if remaining != "" {
		out = append(out, remaining)
	}
	return out
}

func bestWrapIndex(line string, width int) int {
	best := 0
	visible := 0
	seenNonSpace := false
	for i, r := range line {
		visible++
		if visible > width {
			break
		}
		if r != ' ' {
			seenNonSpace = true
		}
		if seenNonSpace && r == ' ' {
			best = i + len(string(r))
		}
	}
	if best > 0 {
		return best
	}
	index := 0
	visible = 0
	for i, r := range line {
		visible++
		if visible > width {
			break
		}
		index = i + len(string(r))
	}
	if index <= 0 {
		return len(line)
	}
	return index
}

func fitPaneContent(content string, width int) string {
	if width <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = chansi.Truncate(line, width, "…")
	}
	return strings.Join(lines, "\n")
}

var (
	headerStyle      = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	panelStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	titleStyle       = lipgloss.NewStyle().Bold(true).Underline(true)
	subtitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	descriptionStyle = lipgloss.NewStyle().Faint(true)
	commandStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("0")).Padding(0, 1)
	missingTagStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	existingTagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	footerStyle      = lipgloss.NewStyle().Padding(0, 1)
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
)

func statusStyle(status workflow.Status) string {
	color := lipgloss.Color("7")
	switch status {
	case workflow.StatusReady:
		color = lipgloss.Color("10")
	case workflow.StatusNeedsConfirmation:
		color = lipgloss.Color("11")
	case workflow.StatusBlocked:
		color = lipgloss.Color("8")
	case workflow.StatusCompleted:
		color = lipgloss.Color("12")
	case workflow.StatusSkipped, workflow.StatusExternallySatisfied:
		color = lipgloss.Color("13")
	case workflow.StatusFailed:
		color = lipgloss.Color("9")
	}
	return lipgloss.NewStyle().Foreground(color).Render("[" + string(status) + "]")
}

func paddedStatusStyle(status workflow.Status, width int) string {
	label := statusStyle(status)
	visible := chansi.StringWidth(label)
	if visible >= width {
		return chansi.Truncate(label, width, "…")
	}
	return label + strings.Repeat(" ", width-visible)
}

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
)

// chainNames maps chain IDs to human-readable names
var chainNames = map[string]string{
	"11155420": "OP Sepolia",
	"1301":     "Unichain Sepolia",
	"10":       "OP Mainnet",
	"1":        "Ethereum",
	"8453":     "Base",
	"84532":    "Base Sepolia",
}

// chainExplorers maps chain IDs to Blockscout explorer base URLs
var chainExplorers = map[string]string{
	"11155420": "https://optimism-sepolia.blockscout.com",
	"1301":     "https://unichain-sepolia.blockscout.com",
	"10":       "https://optimism.blockscout.com",
	"8453":     "https://base.blockscout.com",
	"84532":    "https://base-sepolia.blockscout.com",
}

// RecentQuery matches the spammer's RecentQuery struct
// Note: ChainID is string because eth.ChainID serializes to decimal string via MarshalText
type RecentQuery struct {
	Timestamp   string `json:"timestamp"`
	ChainID     string `json:"chain_id"`
	BlockNumber uint64 `json:"block_number"`
	TxHash      string `json:"tx_hash"`
	LogIndex    uint   `json:"log_index"`
	Address     string `json:"address"`
	QueryType   string `json:"query_type"`
	Result      string `json:"result"`
}

var (
	FilterMetricsFlag = &cli.StringFlag{
		Name:    "filter-metrics",
		Usage:   "Filter service metrics endpoint",
		Value:   "http://localhost:7300/metrics",
		EnvVars: []string{"FILTER_METRICS"},
	}
	SpammerMetricsFlag = &cli.StringFlag{
		Name:    "spammer-metrics",
		Usage:   "Spammer metrics endpoints (comma-separated for multiple chains)",
		Value:   "http://localhost:7301/metrics",
		EnvVars: []string{"SPAMMER_METRICS"},
	}
	RefreshIntervalFlag = &cli.StringFlag{
		Name:    "refresh",
		Usage:   "Refresh interval (e.g., 1s, 2s)",
		Value:   "2s",
		EnvVars: []string{"REFRESH_INTERVAL"},
	}
)

type Metrics struct {
	// Filter metrics
	FilterUp           float64
	FilterFailsafe     float64
	FilterChainHead    map[string]float64
	FilterBackfillProg map[string]float64 // 1.0 = ready
	FilterCheckSuccess float64
	FilterCheckFailed  float64
	FilterReorgs       map[string]float64

	// Chain ingestion metrics (per chain)
	BlocksSealed map[string]float64
	LogsAdded    map[string]float64

	// Spammer metrics
	SpammerUp              float64
	SpammerValidAccepted   float64
	SpammerValidRejected   float64
	SpammerInvalidAccepted float64
	SpammerInvalidRejected float64
	SpammerErrors          float64
	SpammerLatencyValid    float64
	SpammerLatencyInvalid  float64
}

func main() {
	app := cli.NewApp()
	app.Name = "filter-dashboard"
	app.Usage = "Terminal dashboard for op-interop-filter observability"
	app.Flags = []cli.Flag{
		FilterMetricsFlag,
		SpammerMetricsFlag,
		RefreshIntervalFlag,
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cliCtx *cli.Context) error {
	filterURL := cliCtx.String(FilterMetricsFlag.Name)
	spammerURLsStr := cliCtx.String(SpammerMetricsFlag.Name)
	refreshStr := cliCtx.String(RefreshIntervalFlag.Name)

	// Parse comma-separated spammer URLs
	spammerURLs := strings.Split(spammerURLsStr, ",")
	for i := range spammerURLs {
		spammerURLs[i] = strings.TrimSpace(spammerURLs[i])
	}

	refresh, err := time.ParseDuration(refreshStr)
	if err != nil {
		return fmt.Errorf("invalid refresh interval: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	startTime := time.Now()
	ticker := time.NewTicker(refresh)
	defer ticker.Stop()

	// Initial render
	render(filterURL, spammerURLs, startTime)

	for {
		select {
		case <-ctx.Done():
			clearScreen()
			fmt.Println("Dashboard stopped.")
			return nil
		case <-ticker.C:
			render(filterURL, spammerURLs, startTime)
		}
	}
}

func render(filterURL string, spammerURLs []string, startTime time.Time) {
	clearScreen()

	m := fetchAllMetrics(filterURL, spammerURLs)
	uptime := time.Since(startTime).Round(time.Second)

	// Header
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	printLine("           OP-INTEROP-FILTER DASHBOARD")
	printLine(fmt.Sprintf("  Uptime: %-20s                            %s", uptime, time.Now().Format("15:04:05")))
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")

	// Filter Service Status
	filterStatus := "[DOWN]"
	if m.FilterUp > 0 {
		filterStatus = "[UP]"
	}
	failsafeStatus := "[OK]"
	if m.FilterFailsafe > 0 {
		failsafeStatus = "[TRIGGERED]"
	}

	printLine("  FILTER SERVICE")
	printLine(fmt.Sprintf("    Status: %-10s  Failsafe: %-12s", filterStatus, failsafeStatus))

	// Chain status (sorted for consistent display)
	if len(m.FilterChainHead) > 0 {
		chainIDs := make([]string, 0, len(m.FilterChainHead))
		for chainID := range m.FilterChainHead {
			chainIDs = append(chainIDs, chainID)
		}
		sort.Strings(chainIDs)

		for _, chainID := range chainIDs {
			head := m.FilterChainHead[chainID]
			// Chain is ready when backfill_progress >= 1.0
			ready := "..."
			backfill := 0.0
			if bf, ok := m.FilterBackfillProg[chainID]; ok {
				backfill = bf * 100
				if bf >= 1.0 {
					ready = "[ready]"
				}
			}
			reorgs := 0.0
			if r, ok := m.FilterReorgs[chainID]; ok {
				reorgs = r
			}
			name := chainName(chainID)
			printLine(fmt.Sprintf("    %-18s %-7s Head=%-10.0f Backfill=%5.1f%% Reorgs=%.0f",
				name, ready, head, backfill, reorgs))
		}
	} else {
		printLine("    No chain data available")
	}

	// Total requests
	totalChecks := m.FilterCheckSuccess + m.FilterCheckFailed
	printLine(fmt.Sprintf("    Requests: %.0f total", totalChecks))

	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")

	// Chain ingestion stats (sorted for consistent display)
	printLine("  INGESTION")
	if len(m.BlocksSealed) > 0 {
		chainIDs := make([]string, 0, len(m.BlocksSealed))
		for chainID := range m.BlocksSealed {
			chainIDs = append(chainIDs, chainID)
		}
		sort.Strings(chainIDs)

		for _, chainID := range chainIDs {
			blocksSealed := m.BlocksSealed[chainID]
			logsAdded := m.LogsAdded[chainID]
			name := chainName(chainID)
			printLine(fmt.Sprintf("    %-18s Blocks=%-10.0f Logs=%-10.0f",
				name, blocksSealed, logsAdded))
		}
	} else {
		printLine("    No ingestion data available")
	}

	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")

	// Spammer Status
	spammerStatus := "[DOWN]"
	if m.SpammerUp > 0 {
		spammerStatus = "[UP]"
	}

	printLine("  SPAMMER")
	printLine(fmt.Sprintf("    Status: %-10s  Errors: %.0f", spammerStatus, m.SpammerErrors))

	// Query stats
	validTotal := m.SpammerValidAccepted + m.SpammerValidRejected
	invalidTotal := m.SpammerInvalidAccepted + m.SpammerInvalidRejected
	allQueries := validTotal + invalidTotal

	printLine(fmt.Sprintf("    Valid queries:   %-6.0f (accepted: %.0f, rejected: %.0f)",
		validTotal, m.SpammerValidAccepted, m.SpammerValidRejected))
	printLine(fmt.Sprintf("    Invalid queries: %-6.0f (accepted: %.0f, rejected: %.0f)",
		invalidTotal, m.SpammerInvalidAccepted, m.SpammerInvalidRejected))

	// Calculate correctness
	correct := m.SpammerValidAccepted + m.SpammerInvalidRejected
	correctRate := 0.0
	if allQueries > 0 {
		correctRate = (correct / allQueries) * 100
	}

	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")
	printLine("  SUMMARY")
	printLine(fmt.Sprintf("    Total Queries: %-8.0f  Correctness: %6.2f%%", allQueries, correctRate))

	// Progress bar for correctness
	barWidth := 40
	filled := int(correctRate / 100 * float64(barWidth))
	bar := strings.Repeat("=", filled) + strings.Repeat("-", barWidth-filled)
	printLine(fmt.Sprintf("    [%s]", bar))

	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")

	// Recent queries - grouped by chain
	printLine("  RECENT QUERIES (newest first)")

	// Fetch queries from all spammer endpoints
	recentQueryList := fetchRecentQueriesFromAll(spammerURLs)

	// Group by chain ID (string since eth.ChainID serializes to decimal string)
	queriesByChain := make(map[string][]RecentQuery)
	for _, q := range recentQueryList {
		queriesByChain[q.ChainID] = append(queriesByChain[q.ChainID], q)
	}

	if len(queriesByChain) > 0 {
		// Sort chain IDs for consistent display
		chainIDs := make([]string, 0, len(queriesByChain))
		for chainID := range queriesByChain {
			chainIDs = append(chainIDs, chainID)
		}
		sort.Strings(chainIDs)

		for _, chainID := range chainIDs {
			queries := queriesByChain[chainID]
			name := chainName(chainID)
			printLine(fmt.Sprintf("  --- %s ---", name))

			// Show up to 3 per chain
			limit := 3
			if len(queries) < limit {
				limit = len(queries)
			}
			for i := 0; i < limit; i++ {
				q := queries[i]
				icon := "OK"
				if (q.QueryType == "valid" && q.Result == "rejected") ||
					(q.QueryType == "invalid" && q.Result == "accepted") {
					icon = "!!"
				}
				// Shorten tx hash for display
				txShort := q.TxHash
				if len(txShort) > 18 {
					txShort = txShort[:10] + "..." + txShort[len(txShort)-6:]
				}
				printLine(fmt.Sprintf("  [%s] %s blk=%-8d log=%-3d tx=%s",
					icon, q.Timestamp, q.BlockNumber, q.LogIndex, txShort))
				// Show address and result on second line
				addrShort := q.Address
				if len(addrShort) > 22 {
					addrShort = addrShort[:12] + "..." + addrShort[len(addrShort)-8:]
				}
				printLine(fmt.Sprintf("       addr=%s type=%-7s result=%-8s",
					addrShort, q.QueryType, q.Result))
				// Show explorer link on third line
				if explorer, ok := chainExplorers[q.ChainID]; ok {
					printLine(fmt.Sprintf("       %s/tx/%s", explorer, q.TxHash))
				}
			}
		}
	} else {
		printLine("    No recent queries")
	}

	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println("Press Ctrl+C to exit")
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// boxWidth is the inner width of the dashboard box (between the ║ characters)
const boxWidth = 76

// printLine prints a line with proper padding to align the right border
// It accounts for emoji display width (emojis display as 2 chars but are 1 rune)
func printLine(content string) {
	// Count display width (emojis count as 2)
	displayWidth := 0
	for _, r := range content {
		if r > 0x1F600 || (r >= 0x2600 && r <= 0x27BF) || (r >= 0x1F300 && r <= 0x1F9FF) {
			displayWidth += 2 // emoji takes 2 display columns
		} else {
			displayWidth += 1
		}
	}

	padding := boxWidth - displayWidth
	if padding < 0 {
		padding = 0
	}
	fmt.Printf("║%s%s║\n", content, strings.Repeat(" ", padding))
}

func fetchAllMetrics(filterURL string, spammerURLs []string) Metrics {
	m := Metrics{
		FilterChainHead:    make(map[string]float64),
		FilterBackfillProg: make(map[string]float64),
		FilterReorgs:       make(map[string]float64),
		BlocksSealed:       make(map[string]float64),
		LogsAdded:          make(map[string]float64),
	}

	// Fetch filter metrics
	filterMetrics := fetchMetrics(filterURL)
	m.FilterUp = filterMetrics["op_interop_filter_default_up"]
	m.FilterFailsafe = filterMetrics["op_interop_filter_default_failsafe_enabled"]
	m.FilterCheckSuccess = filterMetrics["op_interop_filter_default_check_access_list_total{success=\"true\"}"]
	m.FilterCheckFailed = filterMetrics["op_interop_filter_default_check_access_list_total{success=\"false\"}"]

	// Parse chain-specific metrics
	for k, v := range filterMetrics {
		if strings.HasPrefix(k, "op_interop_filter_default_chain_head{") {
			chainID := extractLabel(k, "chain_id")
			m.FilterChainHead[chainID] = v
		}
		if strings.HasPrefix(k, "op_interop_filter_default_backfill_progress{") {
			chainID := extractLabel(k, "chain_id")
			m.FilterBackfillProg[chainID] = v
		}
		if strings.HasPrefix(k, "op_interop_filter_default_reorg_detected_total{") {
			chainID := extractLabel(k, "chain_id")
			m.FilterReorgs[chainID] = v
		}
		// Chain ingestion metrics (actual metric names without logsdb_ prefix)
		if strings.HasPrefix(k, "op_interop_filter_default_blocks_sealed_total{") {
			chainID := extractLabel(k, "chain_id")
			m.BlocksSealed[chainID] = v
		}
		if strings.HasPrefix(k, "op_interop_filter_default_logs_added_total{") {
			chainID := extractLabel(k, "chain_id")
			m.LogsAdded[chainID] = v
		}
	}

	// Fetch spammer metrics from all spammer endpoints and aggregate
	for _, spammerURL := range spammerURLs {
		spammerMetrics := fetchMetrics(spammerURL)
		// Take the max of "up" values (any spammer up = spammer up)
		if spammerMetrics["filter_spammer_up"] > m.SpammerUp {
			m.SpammerUp = spammerMetrics["filter_spammer_up"]
		}
		// Sum errors and query counts across all spammers
		m.SpammerErrors += spammerMetrics["filter_spammer_errors_total"]
		m.SpammerValidAccepted += findMetricWithLabels(spammerMetrics, "filter_spammer_queries_total{",
			map[string]string{"type": "valid", "result": "accepted"})
		m.SpammerValidRejected += findMetricWithLabels(spammerMetrics, "filter_spammer_queries_total{",
			map[string]string{"type": "valid", "result": "rejected"})
		m.SpammerInvalidAccepted += findMetricWithLabels(spammerMetrics, "filter_spammer_queries_total{",
			map[string]string{"type": "invalid", "result": "accepted"})
		m.SpammerInvalidRejected += findMetricWithLabels(spammerMetrics, "filter_spammer_queries_total{",
			map[string]string{"type": "invalid", "result": "rejected"})
	}

	return m
}

func fetchMetrics(url string) map[string]float64 {
	result := make(map[string]float64)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return result
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Parse prometheus text format: metric_name{labels} value
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		metricName := parts[0]
		value, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}

		result[metricName] = value
	}

	return result
}

func extractLabel(metric, labelName string) string {
	// Extract label value from metric like: metric_name{chain_id="123"}
	start := strings.Index(metric, labelName+"=\"")
	if start == -1 {
		return ""
	}
	start += len(labelName) + 2
	end := strings.Index(metric[start:], "\"")
	if end == -1 {
		return ""
	}
	return metric[start : start+end]
}

// chainName returns a human-readable name for a chain ID
func chainName(chainID string) string {
	if name, ok := chainNames[chainID]; ok {
		return name
	}
	return "Chain " + chainID
}

// hasLabels checks if a metric has the specified label values (order-independent)
func hasLabels(metric string, labels map[string]string) bool {
	for k, v := range labels {
		if extractLabel(metric, k) != v {
			return false
		}
	}
	return true
}

// findMetricWithLabels finds a metric value with specific labels
func findMetricWithLabels(metrics map[string]float64, prefix string, labels map[string]string) float64 {
	for k, v := range metrics {
		if strings.HasPrefix(k, prefix) && hasLabels(k, labels) {
			return v
		}
	}
	return 0
}

func fetchRecentQueries(spammerURL string) []RecentQuery {
	// Convert metrics URL to recent URL (replace /metrics with /recent)
	recentURL := strings.Replace(spammerURL, "/metrics", "/recent", 1)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(recentURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var queries []RecentQuery
	if err := json.NewDecoder(resp.Body).Decode(&queries); err != nil {
		return nil
	}
	return queries
}

func fetchRecentQueriesFromAll(spammerURLs []string) []RecentQuery {
	var allQueries []RecentQuery
	for _, url := range spammerURLs {
		queries := fetchRecentQueries(url)
		allQueries = append(allQueries, queries...)
	}
	return allQueries
}

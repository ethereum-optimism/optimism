package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/urfave/cli/v2"
)

const (
	serviceStateRunning    = "running"
	serviceStateStopping   = "stopping"
	serviceStateStopped    = "stopped"
	serviceStateStarting   = "starting"
	serviceStateError      = "error"
	serviceStateStopFailed = "stop_failed"

	serviceHealthOK          = "ok"
	serviceHealthUnreachable = "unreachable"
	serviceHealthStopped     = "stopped"
	serviceHealthUnknown     = "unknown"

	controlStopTimeout   = 30 * time.Second
	controlStartTimeout  = 30 * time.Second
	controlStartStuck    = 2 * time.Minute
	controlHealthTimeout = time.Second
)

var errControlOperationPending = errors.New("control operation still in progress")

var controlSessionFlag = &cli.StringFlag{
	Name:  "session",
	Usage: "op-up control session id. Defaults to the newest live session.",
}

type controlledService struct {
	ID          string
	Name        string
	Kind        string
	ChainID     eth.ChainID
	Endpoint    *devnetEndpoint
	EndpointURL string
	RPC         opclient.RPC
	Control     stack.ControlledLifecycle
	LogDir      string

	mu            sync.Mutex
	state         string
	lastOperation string
	lastError     string
	lastUpdated   time.Time
	opSeq         uint64
}

type controlledServiceStatus struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Chain         string `json:"chain"`
	State         string `json:"state"`
	Health        string `json:"health"`
	EndpointURL   string `json:"endpoint_url,omitempty"`
	LogDir        string `json:"log_dir,omitempty"`
	LastOperation string `json:"last_operation,omitempty"`
	LastError     string `json:"last_error,omitempty"`
	LastUpdated   string `json:"last_updated,omitempty"`
}

type controlSessionMetadata struct {
	ID         string                    `json:"id"`
	Preset     string                    `json:"preset"`
	PID        int                       `json:"pid"`
	StartedAt  string                    `json:"started_at"`
	ControlURL string                    `json:"control_url"`
	Token      string                    `json:"token"`
	LogFile    string                    `json:"log_file,omitempty"`
	Services   []controlledServiceStatus `json:"services"`
}

type controlServer struct {
	session controlSessionMetadata
	token   string
	byID    map[string]*controlledService
	server  *http.Server
	file    string
}

type controlResponse struct {
	Session  controlSessionMetadata    `json:"session"`
	Service  *controlledServiceStatus  `json:"service,omitempty"`
	Services []controlledServiceStatus `json:"services,omitempty"`
	Error    string                    `json:"error,omitempty"`
}

func controlCommand() *cli.Command {
	flags := []cli.Flag{controlSessionFlag, dirFlag}
	return &cli.Command{
		Name:  "control",
		Usage: "control services in a running op-up devnet",
		Subcommands: []*cli.Command{
			{
				Name:    "services",
				Aliases: []string{"ls", "list", "ps"},
				Usage:   "list controllable services",
				Flags:   flags,
				Action: func(cliCtx *cli.Context) error {
					return runControlCLI(cliCtx.Context, cliCtx, "services", "")
				},
			},
			{
				Name:      "status",
				Usage:     "show one service status",
				ArgsUsage: "<service-id>",
				Flags:     flags,
				Action: func(cliCtx *cli.Context) error {
					id, err := oneControlServiceArg(cliCtx)
					if err != nil {
						return err
					}
					return runControlCLI(cliCtx.Context, cliCtx, "status", id)
				},
			},
			controlActionCommand("start"),
			controlActionCommand("stop"),
			controlActionCommand("restart"),
		},
	}
}

func controlActionCommand(action string) *cli.Command {
	return &cli.Command{
		Name:      action,
		Usage:     action + " one service",
		ArgsUsage: "<service-id>",
		Flags:     []cli.Flag{controlSessionFlag, dirFlag},
		Action: func(cliCtx *cli.Context) error {
			id, err := oneControlServiceArg(cliCtx)
			if err != nil {
				return err
			}
			return runControlCLI(cliCtx.Context, cliCtx, action, id)
		},
	}
}

func oneControlServiceArg(cliCtx *cli.Context) (string, error) {
	if cliCtx.NArg() != 1 {
		return "", fmt.Errorf("expected exactly one service id")
	}
	return cliCtx.Args().First(), nil
}

func runControlCLI(ctx context.Context, cliCtx *cli.Context, action string, serviceID string) error {
	dir := cliPath(cliCtx, dirFlag.Name)
	sessionID := cliString(cliCtx, controlSessionFlag.Name)
	session, err := selectControlSession(dir, sessionID)
	if err != nil {
		return err
	}
	resp, err := callControlServer(ctx, session, action, serviceID)
	if err != nil {
		return err
	}
	printControlResponse(cliCtx.App.Writer, resp, action)
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func selectControlSession(dir string, sessionID string) (controlSessionMetadata, error) {
	controlDir := filepath.Join(dir, "control")
	files, err := os.ReadDir(controlDir)
	if err != nil {
		return controlSessionMetadata{}, fmt.Errorf("read control sessions: %w", err)
	}
	var candidates []controlSessionMetadata
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		path := filepath.Join(controlDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var meta controlSessionMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		if sessionID != "" && meta.ID != sessionID {
			continue
		}
		if !controlSessionLive(meta) {
			continue
		}
		candidates = append(candidates, meta)
	}
	if len(candidates) == 0 {
		if sessionID != "" {
			return controlSessionMetadata{}, fmt.Errorf("no live op-up session %q found", sessionID)
		}
		return controlSessionMetadata{}, fmt.Errorf("no live op-up sessions found")
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].StartedAt > candidates[j].StartedAt
	})
	return candidates[0], nil
}

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func controlSessionLive(meta controlSessionMetadata) bool {
	if !processExists(meta.PID) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(meta.ControlURL, "/")+"/healthz", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+meta.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func callControlServer(ctx context.Context, session controlSessionMetadata, action string, serviceID string) (controlResponse, error) {
	method := http.MethodGet
	path := "/services"
	if action != "services" {
		path = "/services/" + serviceID
		if action != "status" {
			method = http.MethodPost
			path += "/" + action
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(session.ControlURL, "/")+path, nil)
	if err != nil {
		return controlResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+session.Token)
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return controlResponse{}, err
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return controlResponse{}, err
	}
	var resp controlResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return controlResponse{}, fmt.Errorf("decode control response: %w", err)
	}
	if httpResp.StatusCode >= 400 && resp.Error == "" {
		resp.Error = httpResp.Status
	}
	return resp, nil
}

func printControlResponse(w io.Writer, resp controlResponse, action string) {
	fmt.Fprintf(w, "Session: %s\nPreset: %s\nPID: %d\n", resp.Session.ID, resp.Session.Preset, resp.Session.PID)
	if resp.Session.LogFile != "" {
		fmt.Fprintf(w, "Logs: %s\n", resp.Session.LogFile)
	}
	fmt.Fprintln(w)
	if resp.Service != nil {
		printControlServices(w, []controlledServiceStatus{*resp.Service})
		if resp.Error == "" && action != "status" && action != "services" && resp.Service.LastError != "" && (resp.Service.State == serviceStateStarting || resp.Service.State == serviceStateStopping) {
			fmt.Fprintf(w, "\nOperation still in progress for %s; rerun `op-up control status %s --session %s`.\n", resp.Service.ID, resp.Service.ID, resp.Session.ID)
		}
		return
	}
	printControlServices(w, resp.Services)
}

func printControlServices(w io.Writer, services []controlledServiceStatus) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	showLogDir := false
	for _, svc := range services {
		if svc.LogDir != "" {
			showLogDir = true
			break
		}
	}
	if showLogDir {
		fmt.Fprintln(tw, "SERVICE ID\tKIND\tCHAIN\tSTATE\tHEALTH\tENDPOINT\tLOG DIR\tLAST ERROR")
	} else {
		fmt.Fprintln(tw, "SERVICE ID\tKIND\tCHAIN\tSTATE\tHEALTH\tENDPOINT\tLAST ERROR")
	}
	for _, svc := range services {
		if showLogDir {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", svc.ID, svc.Kind, svc.Chain, svc.State, svc.Health, svc.EndpointURL, svc.LogDir, svc.LastError)
		} else {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", svc.ID, svc.Kind, svc.Chain, svc.State, svc.Health, svc.EndpointURL, svc.LastError)
		}
	}
	_ = tw.Flush()
}

func newControlServer(ctx context.Context, cfg opUpConfig, spec *devnetPreset, services []*controlledService, runID string, logFilePath string) (*controlServer, error) {
	if len(services) == 0 {
		return nil, nil
	}
	controlDir := filepath.Join(cfg.Dir, "control")
	if err := os.MkdirAll(controlDir, 0o700); err != nil {
		return nil, fmt.Errorf("create control dir: %w", err)
	}
	if err := os.Chmod(controlDir, 0o700); err != nil {
		return nil, fmt.Errorf("restrict control dir: %w", err)
	}
	token, err := randomControlToken()
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for control server: %w", err)
	}
	id := runID
	if id == "" {
		id = newOpUpRunID(spec.Name)
	}
	startedAt := time.Now().UTC().Format(time.RFC3339Nano)
	byID := make(map[string]*controlledService, len(services))
	for _, svc := range services {
		svc.initState()
		byID[svc.ID] = svc
	}
	session := controlSessionMetadata{
		ID:         id,
		Preset:     spec.Name,
		PID:        os.Getpid(),
		StartedAt:  startedAt,
		ControlURL: "http://" + listener.Addr().String(),
		Token:      token,
		LogFile:    logFilePath,
		Services:   serviceStatuses(ctx, services),
	}
	file := filepath.Join(controlDir, id+".json")
	if err := writeControlSessionFile(file, session); err != nil {
		_ = listener.Close()
		return nil, err
	}
	cs := &controlServer{session: session, token: token, byID: byID, file: file}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", cs.handleHealthz)
	mux.HandleFunc("/services", cs.handleServices)
	mux.HandleFunc("/services/", cs.handleService)
	cs.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		_ = cs.server.Serve(listener)
	}()
	go func() {
		<-ctx.Done()
		_ = cs.Close()
	}()
	return cs, nil
}

func (c *controlServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if !c.authorized(w, r) {
		return
	}
	if r.URL.Path != "/healthz" || r.Method != http.MethodGet {
		writeControlError(w, http.StatusNotFound, "not found")
		return
	}
	session := c.session
	session.Services = nil
	writeControlJSON(w, http.StatusOK, controlResponse{Session: session})
}

func writeControlSessionFile(path string, session controlSessionMetadata) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal control session: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write control session: %w", err)
	}
	return nil
}

func (c *controlServer) Close() error {
	if c == nil {
		return nil
	}
	_ = os.Remove(c.file)
	if c.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.server.Shutdown(ctx)
}

func (c *controlServer) handleServices(w http.ResponseWriter, r *http.Request) {
	if !c.authorized(w, r) {
		return
	}
	if r.URL.Path != "/services" || r.Method != http.MethodGet {
		writeControlError(w, http.StatusNotFound, "not found")
		return
	}
	writeControlJSON(w, http.StatusOK, controlResponse{Session: c.currentSession(r.Context()), Services: c.statuses(r.Context())})
}

func (c *controlServer) handleService(w http.ResponseWriter, r *http.Request) {
	if !c.authorized(w, r) {
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/services/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		c.writeControlError(w, r, http.StatusNotFound, "missing service id")
		return
	}
	svc, ok := c.byID[parts[0]]
	if !ok {
		c.writeControlError(w, r, http.StatusNotFound, "unknown service id "+parts[0])
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		status := svc.status(r.Context())
		writeControlJSON(w, http.StatusOK, controlResponse{Session: c.sessionSummary(), Service: &status})
		return
	}
	if len(parts) != 2 || r.Method != http.MethodPost {
		c.writeControlError(w, r, http.StatusNotFound, "not found")
		return
	}
	var err error
	switch parts[1] {
	case "start":
		err = svc.start(r.Context())
	case "stop":
		err = svc.stop(r.Context())
	case "restart":
		err = svc.restart(r.Context())
	default:
		c.writeControlError(w, r, http.StatusNotFound, "unknown action "+parts[1])
		return
	}
	status := svc.status(r.Context())
	code := http.StatusOK
	resp := controlResponse{Session: c.sessionSummary(), Service: &status}
	if err != nil {
		if errors.Is(err, errControlOperationPending) {
			code = http.StatusAccepted
		} else {
			code = http.StatusConflict
			resp.Error = err.Error()
		}
	}
	writeControlJSON(w, code, resp)
}

func (c *controlServer) writeControlError(w http.ResponseWriter, r *http.Request, code int, message string) {
	writeControlJSON(w, code, controlResponse{Session: c.sessionSummary(), Error: message})
}

func (c *controlServer) authorized(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Authorization") != "Bearer "+c.token {
		writeControlError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	return true
}

func (c *controlServer) currentSession(ctx context.Context) controlSessionMetadata {
	session := c.session
	session.Services = c.statuses(ctx)
	return session
}

func (c *controlServer) sessionSummary() controlSessionMetadata {
	session := c.session
	session.Services = nil
	return session
}

func (c *controlServer) statuses(ctx context.Context) []controlledServiceStatus {
	services := make([]*controlledService, 0, len(c.byID))
	for _, svc := range c.byID {
		services = append(services, svc)
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].ID < services[j].ID
	})
	return serviceStatuses(ctx, services)
}

func serviceStatuses(ctx context.Context, services []*controlledService) []controlledServiceStatus {
	out := make([]controlledServiceStatus, 0, len(services))
	for _, svc := range services {
		out = append(out, svc.status(ctx))
	}
	return out
}

func (s *controlledService) initState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Control != nil && s.Control.Running() {
		s.state = serviceStateRunning
	} else {
		s.state = serviceStateStopped
	}
	s.lastUpdated = time.Now().UTC()
}

func (s *controlledService) status(ctx context.Context) controlledServiceStatus {
	s.mu.Lock()
	state := s.state
	lastOperation := s.lastOperation
	lastError := s.lastError
	lastUpdated := s.lastUpdated
	s.mu.Unlock()
	endpointURL := ""
	if s.EndpointURL != "" {
		endpointURL = s.EndpointURL
	}
	chain := "shared"
	if s.ChainID != (eth.ChainID{}) {
		chain = s.ChainID.String()
	}
	updated := ""
	if !lastUpdated.IsZero() {
		updated = lastUpdated.Format(time.RFC3339Nano)
	}
	return controlledServiceStatus{
		ID:            s.ID,
		Name:          s.Name,
		Kind:          s.Kind,
		Chain:         chain,
		State:         state,
		Health:        s.health(ctx, state),
		EndpointURL:   endpointURL,
		LogDir:        s.LogDir,
		LastOperation: lastOperation,
		LastError:     lastError,
		LastUpdated:   updated,
	}
}

func (s *controlledService) health(ctx context.Context, state string) string {
	if state == serviceStateStopped || state == serviceStateStopFailed {
		return serviceHealthStopped
	}
	if s.RPC == nil {
		return serviceHealthUnknown
	}
	callCtx, cancel := context.WithTimeout(ctx, controlHealthTimeout)
	defer cancel()
	var out json.RawMessage
	if err := s.RPC.CallContext(callCtx, &out, "rpc_modules"); err != nil {
		return serviceHealthUnreachable
	}
	return serviceHealthOK
}

func (s *controlledService) start(ctx context.Context) error {
	if s.Control == nil {
		return fmt.Errorf("service %s is not controllable", s.ID)
	}
	s.mu.Lock()
	if s.state != serviceStateStopped {
		err := fmt.Errorf("service %s cannot start from state %s", s.ID, s.state)
		s.mu.Unlock()
		return err
	}
	s.opSeq++
	seq := s.opSeq
	s.state = serviceStateStarting
	s.lastOperation = "start"
	s.lastError = ""
	s.lastUpdated = time.Now().UTC()
	s.mu.Unlock()

	opCtx, cancel := context.WithTimeout(ctx, controlStartTimeout)
	done := make(chan error, 1)
	go func() {
		done <- s.Control.StartControlled(opCtx)
	}()
	select {
	case err := <-done:
		cancel()
		return s.completeStart(seq, err)
	case <-opCtx.Done():
		err := fmt.Errorf("start still in progress after %s: %w", controlStartTimeout, opCtx.Err())
		s.noteStartPending(seq, err)
		go func() {
			timer := time.NewTimer(controlStartStuck)
			defer timer.Stop()
			select {
			case err := <-done:
				_ = s.completeStart(seq, err)
			case <-timer.C:
				s.noteStartPending(seq, fmt.Errorf("start has not completed after %s; safest recovery is restarting the op-up devnet", controlStartStuck))
				_ = s.completeStart(seq, <-done)
			}
		}()
		cancel()
		return errControlOperationPending
	}
}

func (s *controlledService) noteStartPending(seq uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opSeq != seq {
		return
	}
	s.state = serviceStateStarting
	s.lastError = err.Error()
	s.lastUpdated = time.Now().UTC()
}

func (s *controlledService) completeStart(seq uint64, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opSeq != seq {
		return err
	}
	if err != nil {
		s.state = serviceStateError
		s.lastError = err.Error()
	} else {
		s.state = serviceStateRunning
	}
	s.lastUpdated = time.Now().UTC()
	return err
}

func (s *controlledService) stop(ctx context.Context) error {
	if s.Control == nil {
		return fmt.Errorf("service %s is not controllable", s.ID)
	}
	s.mu.Lock()
	switch s.state {
	case serviceStateStopped:
		s.mu.Unlock()
		return nil
	case serviceStateStarting, serviceStateStopping:
		err := fmt.Errorf("service %s is busy in state %s", s.ID, s.state)
		s.mu.Unlock()
		return err
	}
	s.opSeq++
	seq := s.opSeq
	s.state = serviceStateStopping
	s.lastOperation = "stop"
	s.lastError = ""
	s.lastUpdated = time.Now().UTC()
	s.mu.Unlock()

	opCtx, cancel := context.WithTimeout(ctx, controlStopTimeout)
	done := make(chan error, 1)
	go func() {
		done <- s.Control.StopControlled(opCtx)
	}()
	select {
	case err := <-done:
		cancel()
		return s.completeStop(seq, err)
	case <-opCtx.Done():
		err := opCtx.Err()
		go func() {
			if lateErr := <-done; lateErr == nil {
				_ = s.completeStop(seq, nil)
			}
		}()
		cancel()
		return s.completeStop(seq, err)
	}
}

func (s *controlledService) completeStop(seq uint64, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opSeq != seq {
		return err
	}
	if err != nil {
		s.state = serviceStateStopFailed
		s.lastError = err.Error()
	} else {
		s.state = serviceStateStopped
		s.lastError = ""
	}
	s.lastUpdated = time.Now().UTC()
	return err
}

func (s *controlledService) restart(ctx context.Context) error {
	if err := s.stop(ctx); err != nil {
		return err
	}
	return s.start(ctx)
}

func (s *controlledService) proxyAvailable() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == serviceStateRunning
}

func writeControlJSON(w http.ResponseWriter, code int, value controlResponse) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		writeControlError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(append(data, '\n'))
}

func writeControlError(w http.ResponseWriter, code int, message string) {
	writeControlJSON(w, code, controlResponse{Error: message})
}

func randomControlToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate control token: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

func backendStoppedError(id json.RawMessage, method string) rpcProxyResponse {
	return rpcProxyErr(id, -32000, fmt.Sprintf("backend stopped or unavailable for method %q", method))
}

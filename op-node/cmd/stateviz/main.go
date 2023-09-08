package main

import (
	"bufio"
	"compress/gzip"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	ophttp "github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum/go-ethereum/log"
)

var (
	snapshot   = flag.String("snapshot", "", "path to snapshot log")
	listenAddr = flag.String("addr", "", "listen address of webserver")
	refresh    = flag.Duration("refresh", 10*time.Second, "snapshot refresh rate")
)

var (
	entries      map[string][]SnapshotState
	entriesMutex sync.Mutex

	assetFS fs.FS
)

type SnapshotState struct {
	Timestamp       string         `json:"t"`
	EngineAddr      string         `json:"engine_addr"`
	Event           string         `json:"event"`           // event name
	L1Head          eth.L1BlockRef `json:"l1Head"`          // what we see as head on L1
	L1Current       eth.L1BlockRef `json:"l1Current"`       // l1 block that the derivation is currently using
	L2Head          eth.L2BlockRef `json:"l2Head"`          // l2 block that was last optimistically accepted (unsafe head)
	L2Safe          eth.L2BlockRef `json:"l2Safe"`          // l2 block that was last derived
	L2FinalizedHead eth.BlockID    `json:"l2FinalizedHead"` // l2 block that is irreversible
}

func (e *SnapshotState) UnmarshalJSON(data []byte) error {
	t := struct {
		Timestamp       string          `json:"t"`
		EngineAddr      string          `json:"engine_addr"`
		Event           string          `json:"event"`
		L1Head          json.RawMessage `json:"l1Head"`
		L1Current       json.RawMessage `json:"l1Current"`
		L2Head          json.RawMessage `json:"l2Head"`
		L2Safe          json.RawMessage `json:"l2Safe"`
		L2FinalizedHead json.RawMessage `json:"l2FinalizedHead"`
	}{}
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	e.Timestamp = t.Timestamp
	e.EngineAddr = t.EngineAddr
	e.Event = t.Event

	unquote := func(d json.RawMessage) []byte {
		s, _ := strconv.Unquote(string(d))
		return []byte(s)
	}

	if err := json.Unmarshal(unquote(t.L1Head), &e.L1Head); err != nil {
		return err
	}
	if err := json.Unmarshal(unquote(t.L1Current), &e.L1Current); err != nil {
		return err
	}
	if err := json.Unmarshal(unquote(t.L2Head), &e.L2Head); err != nil {
		return err
	}
	if err := json.Unmarshal(unquote(t.L2Safe), &e.L2Safe); err != nil {
		return err
	}
	if err := json.Unmarshal(unquote(t.L2FinalizedHead), &e.L2FinalizedHead); err != nil {
		return err
	}
	return nil
}

//go:embed assets
var embeddedAssets embed.FS

func main() {
	flag.Parse()

	log.Root().SetHandler(
		log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(os.Stdout, log.TerminalFormat(true))),
	)

	if *snapshot == "" {
		log.Crit("missing required -snapshot flag")
	}

	sub, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		log.Crit("Failed to open asset directory", "message", err)
	}
	assetFS = sub

	go func() {
		ticker := time.NewTicker(*refresh)
		defer ticker.Stop()
		for range ticker.C {
			// TODO: incremental load
			log.Info("loading snapshot...")
			if err := loadSnapshot(); err != nil {
				log.Error("failed to load snapshot", "err", err)
			}
		}
	}()

	runServer()
}

func loadSnapshot() error {
	file, err := os.Open(*snapshot)
	if err != nil {
		return fmt.Errorf("%w: failed to open snapshot file", err)
	}
	defer file.Close()

	tempEntries := make(map[string][]SnapshotState)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry SnapshotState
		if err := json.Unmarshal([]byte(scanner.Text()), &entry); err != nil {
			return fmt.Errorf("%w: failed to decode snapshot log", err)
		}

		tempEntries[entry.EngineAddr] = append(tempEntries[entry.EngineAddr], entry)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%w: failed to scan snapshot file", err)
	}

	entriesMutex.Lock()
	entries = tempEntries
	entriesMutex.Unlock()

	return nil
}

func runServer() {
	l, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Crit("Failed to listen on address", "message", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(assetFS)))
	mux.HandleFunc("/logs", makeGzipHandler(logsHandler))

	log.Info("running webserver...")
	httpServer := ophttp.NewHttpServer(mux)
	if err := httpServer.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Crit("http server failed", "message", err)
	}
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func makeGzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	var output [][]SnapshotState

	entriesMutex.Lock()
	// shallow copy so we can update the SnapshotState slice head
	entriesCopy := make(map[string][]SnapshotState)
	for k, v := range entries {
		entriesCopy[k] = v
	}
	entriesMutex.Unlock()

	// sort log entries and zip em up
	// Each record/row contains SnapshotStates for each rollup driver
	// Note that we assume each SnapshotState slice is sorted by the timestamp
	for {
		var min *SnapshotState
		var minKey string
		for k, v := range entriesCopy {
			if len(v) == 0 {
				continue
			}
			if min == nil || v[0].Timestamp < min.Timestamp {
				min = &v[0]
				minKey = k
			}
		}

		if min == nil {
			break
		}

		entriesCopy[minKey] = entriesCopy[minKey][1:]

		rec := make([]SnapshotState, 0, len(entriesCopy))
		rec = append(rec, *min)
		for k, v := range entriesCopy {
			if k != minKey && len(v) != 0 {
				newEntry := v[0]
				newEntry.Timestamp = min.Timestamp
				rec = append(rec, newEntry)
			}
		}
		output = append(output, rec)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=100000")
	if err := json.NewEncoder(w).Encode(output); err != nil {
		log.Warn("failed to encode logs", "message", err)
	}
}

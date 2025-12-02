package oppprof

import (
	"context"
	"io"
	"net"
	"net/http"
	httpPprof "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum/go-ethereum/log"
)

type Service struct {
	listenEnabled bool
	listenAddr    string
	listenPort    int

	profileType     string
	profileDir      string
	profileFilename string

	cpuFile    io.Closer
	httpServer *httputil.HTTPServer
}

func New(listenEnabled bool, listenAddr string, listenPort int, profType profileType, profileDir, profileFilename string) *Service {
	return &Service{
		listenEnabled:   listenEnabled,
		listenAddr:      listenAddr,
		listenPort:      listenPort,
		profileType:     string(profType),
		profileDir:      profileDir,
		profileFilename: profileFilename,
	}
}

func (s *Service) Start() error {
	switch s.profileType {
	case "cpu":
		if err := s.startCPUProfile(); err != nil {
			return err
		}
	case "block":
		runtime.SetBlockProfileRate(1)
	case "mutex":
		runtime.SetMutexProfileFraction(1)
	}
	if s.listenEnabled {
		if err := s.startServer(); err != nil {
			return err
		}
	}
	if s.profileType != "" {
		log.Info("start profiling to file", "profile_type", s.profileType, "profile_filepath", s.buildTargetFilePath())
	}
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	switch s.profileType {
	case "cpu":
		pprof.StopCPUProfile()
		if s.cpuFile != nil {
			if err := s.cpuFile.Close(); err != nil {
				return err
			}
		}
	case "heap":
		runtime.GC()
		fallthrough
	default:
		profile := pprof.Lookup(s.profileType)
		if profile == nil {
			break
		}
		filepath := s.buildTargetFilePath()
		log.Info("saving profile info", "profile_type", s.profileType, "profile_filepath", s.buildTargetFilePath())
		f, err := os.Create(filepath)
		if err != nil {
			return err
		}
		defer f.Close()
		_ = profile.WriteTo(f, 0)
	}
	if s.httpServer != nil {
		if err := s.httpServer.Stop(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) startServer() error {
	log.Debug("starting pprof server", "addr", net.JoinHostPort(s.listenAddr, strconv.Itoa(s.listenPort)))
	mux := http.NewServeMux()

	// have to do below to support multiple servers, since the
	// pprof import only uses DefaultServeMux
	mux.Handle("/debug/pprof/", http.HandlerFunc(httpPprof.Index))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(httpPprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(httpPprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(httpPprof.Trace))

	addr := net.JoinHostPort(s.listenAddr, strconv.Itoa(s.listenPort))

	var err error
	s.httpServer, err = httputil.StartHTTPServer(addr, mux)
	if err != nil {
		return err
	}

	log.Info("started pprof server", "addr", s.httpServer.Addr())
	return nil
}

func (s *Service) startCPUProfile() error {
	f, err := os.Create(s.buildTargetFilePath())
	if err != nil {
		return err
	}
	err = pprof.StartCPUProfile(f)
	s.cpuFile = f
	return err
}

func (s *Service) buildTargetFilePath() string {
	filename := s.profileType + ".prof"
	if s.profileFilename != "" {
		filename = s.profileFilename
	}
	return filepath.Join(s.profileDir, filename)
}

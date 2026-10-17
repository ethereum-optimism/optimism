// Package certman provides live reloading of the certificate and key
// files used by the standard library http.Server. It defines a type,
// certMan, with methods watching and getting the files.
// Only valid certificate and key pairs are loaded and an optional
// logger can be passed to certman for logging providing it implements
// the logger interface.
package certman

import (
	"crypto/tls"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/fsnotify/fsnotify"
)

// ErrAlreadyWatching is returned by Watch when a watcher is already running.
// Stop the current watcher before starting a new one.
var ErrAlreadyWatching = errors.New("certman: already watching")

// A CertMan represents a certificate manager able to watch certificate
// and key pairs for changes.
//
// A CertMan runs at most one watcher goroutine at a time. Watch and Stop are
// the transitions into and out of that state: Watch starts the goroutine, and
// Stop signals it to return and waits for it to release the underlying
// fsnotify watcher. Stop is idempotent, and is a no-op when no watcher is
// running, so it is safe to call before the first Watch, after a failed
// Watch, after a previous Stop, and more than once.
type CertMan struct {
	mu       sync.RWMutex
	certFile string
	keyFile  string
	keyPair  *tls.Certificate
	log      log.Logger

	// watching is closed to signal the watcher goroutine to return. It is nil
	// when no watcher is running, which is what makes Stop idempotent: closing
	// it more than once would panic, and sending on it more than once would
	// block forever once the goroutine is gone.
	// stopped is closed by the watcher goroutine after it has released its
	// resources, so Stop can wait for the shutdown to complete.
	// Both are guarded by mu.
	watching chan struct{}
	stopped  chan struct{}
}

// New creates a new certMan. The certFile and the keyFile
// are both paths to the location of the files. Relative and
// absolute paths are accepted.
func New(logger log.Logger, certFile, keyFile string) (*CertMan, error) {
	var err error
	certFile, err = filepath.Abs(certFile)
	if err != nil {
		return nil, err
	}
	keyFile, err = filepath.Abs(keyFile)
	if err != nil {
		return nil, err
	}
	cm := &CertMan{
		mu:       sync.RWMutex{},
		certFile: certFile,
		keyFile:  keyFile,
		log:      logger,
	}
	return cm, nil
}

// Watch starts watching for changes to the certificate
// and key files. On any change the certificate and key
// are reloaded. If there is an issue the load will fail
// and the old (if any) certificates and keys will continue
// to be used.
//
// Watch returns ErrAlreadyWatching if a watcher is already running; call Stop
// first. If setup fails part-way through, the fsnotify watcher created here is
// closed before the error is returned, so a failed Watch leaves no watcher or
// goroutine behind and a later Watch can still succeed.
func (cm *CertMan) Watch() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("certman: can't create watcher: %w", err)
	}

	certPath := path.Dir(cm.certFile)
	keyPath := path.Dir(cm.keyFile)

	if err = watcher.Add(certPath); err != nil {
		watcher.Close()
		return fmt.Errorf("certman: can't watch %s: %w", certPath, err)
	}
	if keyPath != certPath {
		if err = watcher.Add(keyPath); err != nil {
			watcher.Close()
			return fmt.Errorf("certman: can't watch %s: %w", keyPath, err)
		}
	}
	if err := cm.load(); err != nil {
		cm.log.Error("certman: can't load cert or key file", "err", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.watching != nil {
		watcher.Close()
		return ErrAlreadyWatching
	}
	cm.watching = make(chan struct{})
	cm.stopped = make(chan struct{})
	cm.log.Info("certman: watching for cert and key change")
	go cm.run(watcher, cm.watching, cm.stopped)
	return nil
}

func (cm *CertMan) load() error {
	keyPair, err := tls.LoadX509KeyPair(cm.certFile, cm.keyFile)
	if err == nil {
		cm.mu.Lock()
		cm.keyPair = &keyPair
		cm.mu.Unlock()
		cm.log.Info("certman: certificate and key loaded")
	}
	return err
}

// run watches until watching is closed, then releases the watcher and the
// ticker and closes stopped to report that it is done. It owns watcher for its
// lifetime, so Watch must not touch it after starting this goroutine — that is
// what keeps a second Watch from racing with the first one's shutdown.
func (cm *CertMan) run(watcher *fsnotify.Watcher, watching <-chan struct{}, stopped chan<- struct{}) {
	defer close(stopped)
	defer watcher.Close()
	cm.log.Info("certman: running")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	files := []string{cm.certFile, cm.keyFile}
	reload := time.Time{}

loop:
	for {
		select {
		case <-watching:
			cm.log.Info("watching triggered; break loop")
			break loop
		case <-ticker.C:
			if !reload.IsZero() && time.Now().After(reload) {
				reload = time.Time{}
				cm.log.Info("certman: reloading")
				if err := cm.load(); err != nil {
					cm.log.Error("certman: can't load cert or key file", "err", err)
				}
			}
		case event := <-watcher.Events:
			for _, f := range files {
				if event.Name == f ||
					strings.HasSuffix(event.Name, "/..data") { // kubernetes secrets mount
					// we wait a couple seconds in case the cert and key don't update atomically
					cm.log.Info(fmt.Sprintf("%s was modified, queue reload", f))
					reload = time.Now().Add(2 * time.Second)
				}
			}
		case err := <-watcher.Errors:
			cm.log.Error("certman: error watching files", "err", err)
		}
	}
	cm.log.Info("certman: stopped watching")
}

// GetCertificate returns the loaded certificate for use by
// the GetCertificate field in tls.Config.
func (cm *CertMan) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.keyPair, nil
}

// GetClientCertificate returns the loaded certificate for use by
// the GetClientCertificate field in tls.Config.
func (cm *CertMan) GetClientCertificate(hello *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.keyPair, nil
}

// Stop tells certMan to stop watching for changes to the
// certificate and key files, and returns once the watcher
// goroutine has exited and released the fsnotify watcher.
//
// Stop is a no-op when no watcher is running, and only the first call has an
// effect, so it is safe to call before Watch, after a failed Watch, after a
// previous Stop, and more than once.
func (cm *CertMan) Stop() {
	cm.mu.Lock()
	watching, stopped := cm.watching, cm.stopped
	cm.watching, cm.stopped = nil, nil
	cm.mu.Unlock()
	if watching == nil {
		return
	}
	close(watching)
	<-stopped
}

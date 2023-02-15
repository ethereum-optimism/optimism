// Package certman provides live reloading of the certificate and key
// files used by the standard library http.Server. It defines a type,
// certMan, with methods watching and getting the files.
// Only valid certificate and key pairs are loaded and an optional
// logger can be passed to certman for logging providing it implements
// the logger interface.
package certman

import (
	"crypto/tls"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/fsnotify/fsnotify"
)

// A CertMan represents a certificate manager able to watch certificate
// and key pairs for changes.
type CertMan struct {
	mu       sync.RWMutex
	certFile string
	keyFile  string
	keyPair  *tls.Certificate
	watcher  *fsnotify.Watcher
	watching chan bool
	log      log.Logger
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
func (cm *CertMan) Watch() error {
	var err error
	if cm.watcher, err = fsnotify.NewWatcher(); err != nil {
		return fmt.Errorf("certman: can't create watcher: %w", err)
	}

	certPath := path.Dir(cm.certFile)
	keyPath := path.Dir(cm.keyFile)

	if err = cm.watcher.Add(certPath); err != nil {
		return fmt.Errorf("certman: can't watch %s: %w", certPath, err)
	}
	if keyPath != certPath {
		if err = cm.watcher.Add(keyPath); err != nil {
			return fmt.Errorf("certman: can't watch %s: %w", certPath, err)
		}
	}
	if err := cm.load(); err != nil {
		cm.log.Error("certman: can't load cert or key file", "err", err)
	}
	cm.log.Info("certman: watching for cert and key change")
	cm.watching = make(chan bool)
	go cm.run()
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

func (cm *CertMan) run() {
	cm.log.Info("certman: running")

	ticker := time.NewTicker(2 * time.Second)
	files := []string{cm.certFile, cm.keyFile}
	reload := time.Time{}

loop:
	for {
		select {
		case <-cm.watching:
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
		case event := <-cm.watcher.Events:
			for _, f := range files {
				if event.Name == f ||
					strings.HasSuffix(event.Name, "/..data") { // kubernetes secrets mount
					// we wait a couple seconds in case the cert and key don't update atomically
					cm.log.Info(fmt.Sprintf("%s was modified, queue reload", f))
					reload = time.Now().Add(2 * time.Second)
				}
			}
		case err := <-cm.watcher.Errors:
			cm.log.Error("certman: error watching files", "err", err)
		}
	}
	cm.log.Info("certman: stopped watching")
	cm.watcher.Close()
	ticker.Stop()
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
// certificate and key files.
func (cm *CertMan) Stop() {
	cm.watching <- false
}

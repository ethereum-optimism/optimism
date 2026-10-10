// Copyright 2017 Dyson Simmons. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package certman_test

import (
	"crypto/tls"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/tls/certman"
	"github.com/stretchr/testify/require"
)

// mustNotBlock runs fn in a goroutine and fails if it has not returned within
// the timeout. It is the assertion for "this call must not block", which a
// plain call would only surface as a package-level test timeout.
func mustNotBlock(t *testing.T, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatalf("%s did not return", what)
	}
}

func TestValidPair(t *testing.T) {
	cm, err := certman.New(log.Root(), "./testdata/server1.crt", "./testdata/server1.key")
	if err != nil {
		t.Errorf("could not create certman: %v", err)
	}
	if err := cm.Watch(); err != nil {
		t.Errorf("could not watch files: %v", err)
	}
}

func TestInvalidPair(t *testing.T) {
	cm, err := certman.New(log.Root(), "./testdata/server1.crt", "./testdata/server2.key")
	if err != nil {
		t.Errorf("could not create certman: %v", err)
	}
	if err := cm.Watch(); err != nil {
		t.Errorf("could not watch files: %v", err)
	}
}

func TestCertificateNotFound(t *testing.T) {
	cm, err := certman.New(log.Root(), "./testdata/nothere.crt", "./testdata/server2.key")
	if err != nil {
		t.Errorf("could not create certman: %v", err)
	}
	if err := cm.Watch(); err != nil {
		if !strings.HasPrefix(err.Error(), "certman: can't watch cert file: ") {
			t.Errorf("unexpected watch error: %v", err)
		}
	}
}

func TestKeyNotFound(t *testing.T) {
	cm, err := certman.New(log.Root(), "./testdata/server1.crt", "./testdata/nothere.key")
	if err != nil {
		t.Errorf("could not create certman: %v", err)
	}
	if err := cm.Watch(); err != nil {
		if !strings.HasPrefix(err.Error(), "certman: can't watch key file: ") {
			t.Errorf("unexpected watch error: %v", err)
		}
	}
}

func TestGetCertificate(t *testing.T) {
	cm, err := certman.New(log.Root(), "./testdata/server1.crt", "./testdata/server1.key")
	if err != nil {
		t.Errorf("could not create certman: %v", err)
	}
	if err := cm.Watch(); err != nil {
		t.Errorf("could not watch files: %v", err)
	}
	hello := &tls.ClientHelloInfo{}
	cmCert, err := cm.GetCertificate(hello)
	if err != nil {
		t.Error("could not get certman certificate")
	}
	expectedCert, _ := tls.LoadX509KeyPair("./testdata/server1.crt", "./testdata/server1.key")
	if err != nil {
		t.Errorf("could not load certificate and key files to test: %v", err)
	}
	if !reflect.DeepEqual(cmCert.Certificate, expectedCert.Certificate) {
		t.Errorf("certman certificate doesn't match expected certificate")
	}
}

// TestStopBeforeWatch covers calling Stop before Watch. The watcher channel is
// nil until Watch creates it, so Stop must be a no-op rather than sending on a
// nil channel, which blocks forever.
func TestStopBeforeWatch(t *testing.T) {
	cm, err := certman.New(log.Root(), "./testdata/server1.crt", "./testdata/server1.key")
	require.NoError(t, err)
	mustNotBlock(t, "Stop before Watch", cm.Stop)
}

// TestStopAfterFailedWatch covers the same path reached through a Watch that
// failed to set up: no watcher is running, so Stop must still be a no-op.
func TestStopAfterFailedWatch(t *testing.T) {
	cm, err := certman.New(log.Root(), "./testdata/nonexistent-dir/server1.crt", "./testdata/nonexistent-dir/server1.key")
	require.NoError(t, err)
	require.Error(t, cm.Watch())
	mustNotBlock(t, "Stop after a failed Watch", cm.Stop)
}

// TestDoubleStop covers calling Stop twice. The watcher goroutine exits after
// the first call, so the second call must not send on a channel that no longer
// has a receiver.
func TestDoubleStop(t *testing.T) {
	cm, err := certman.New(log.Root(), "./testdata/server1.crt", "./testdata/server1.key")
	require.NoError(t, err)
	require.NoError(t, cm.Watch())
	mustNotBlock(t, "first Stop", cm.Stop)
	mustNotBlock(t, "second Stop", cm.Stop)
}

// TestDoubleWatch pins down what a second Watch does: it reports
// ErrAlreadyWatching instead of overwriting the channel of the running watcher,
// which would orphan that goroutine and make it unstoppable.
func TestDoubleWatch(t *testing.T) {
	cm, err := certman.New(log.Root(), "./testdata/server1.crt", "./testdata/server1.key")
	require.NoError(t, err)
	require.NoError(t, cm.Watch())
	defer cm.Stop()

	require.ErrorIs(t, cm.Watch(), certman.ErrAlreadyWatching)
	// The first watcher is still the one running, so Stop still terminates it.
	mustNotBlock(t, "Stop after a rejected Watch", cm.Stop)
}

// TestWatchAfterStop covers restarting: Stop clears the watcher state, so a
// later Watch starts a fresh watcher instead of being rejected.
func TestWatchAfterStop(t *testing.T) {
	cm, err := certman.New(log.Root(), "./testdata/server1.crt", "./testdata/server1.key")
	require.NoError(t, err)
	require.NoError(t, cm.Watch())
	cm.Stop()

	require.NoError(t, cm.Watch())
	mustNotBlock(t, "Stop of the restarted watcher", cm.Stop)
}

// TestWatchSetupFailureCleansUp covers a Watch that fails part-way through
// setup: the watcher accepts the cert directory and then rejects the key
// directory. The fsnotify watcher created in Watch must be closed on that path
// rather than leaked, and no watcher goroutine must be left running.
func TestWatchSetupFailureCleansUp(t *testing.T) {
	// The key lives in a directory that does not exist, so the first Add
	// succeeds and the second one fails.
	cm, err := certman.New(log.Root(), "./testdata/server1.crt", "./testdata/nonexistent-dir/server1.key")
	require.NoError(t, err)
	require.Error(t, cm.Watch())
	mustNotBlock(t, "Stop after a partially failed Watch", cm.Stop)
}

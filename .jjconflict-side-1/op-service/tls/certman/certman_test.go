// Copyright 2017 Dyson Simmons. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package certman_test

import (
	"crypto/tls"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/tls/certman"
	"github.com/ethereum/go-ethereum/log"
)

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

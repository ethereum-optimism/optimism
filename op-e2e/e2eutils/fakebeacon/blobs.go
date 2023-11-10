package fakebeacon

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/log"
)

// FakeBeacon presents a beacon-node in testing, without leading any chain-building.
// This merely serves a fake beacon API, and holds on to blocks,
// to complement the actual block-building to happen in testing (e.g. through the fake consensus geth module).
type FakeBeacon struct {
	log log.Logger

	// directory to store blob contents in after the blobs are persisted in a block
	blobsDir  string
	blobsLock sync.Mutex

	beaconSrv         *http.Server
	beaconAPIListener net.Listener

	genesisTime uint64
	blockTime   uint64
}

func NewBeacon(log log.Logger, blobsDir string, genesisTime uint64, blockTime uint64) *FakeBeacon {
	return &FakeBeacon{
		log:         log,
		blobsDir:    blobsDir,
		genesisTime: genesisTime,
		blockTime:   blockTime,
	}
}

func (f *FakeBeacon) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to open tcp listener for http beacon api server: %w", err)
	}
	f.beaconAPIListener = listener

	mux := new(http.ServeMux)
	mux.HandleFunc("/eth/v1/beacon/genesis", func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(&eth.APIGenesisResponse{Data: eth.ReducedGenesisData{GenesisTime: eth.Uint64String(f.genesisTime)}})
		if err != nil {
			f.log.Error("genesis handler err", "err", err)
		}
	})
	mux.HandleFunc("/eth/v1/config/spec", func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(&eth.APIConfigResponse{Data: eth.ReducedConfigData{SecondsPerSlot: eth.Uint64String(f.blockTime)}})
		if err != nil {
			f.log.Error("config handler err", "err", err)
		}
	})
	mux.HandleFunc("/eth/v1/beacon/blob_sidecars/", func(w http.ResponseWriter, r *http.Request) {
		blockID := strings.TrimPrefix(r.URL.Path, "/eth/v1/beacon/blob_sidecars/")
		slot, err := strconv.ParseUint(blockID, 10, 64)
		if err != nil {
			f.log.Error("could not parse block id from request", "url", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bundle, err := f.LoadBlobsBundle(slot)
		if err != nil {
			f.log.Error("failed to load blobs bundle", "slot", slot)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		rawIndices := query["indices"]
		indices := make([]int, 0, len(bundle.Blobs))
		if len(rawIndices) == 0 {
			// request is for all blobs
			for i := range bundle.Blobs {
				indices = append(indices, i)
			}
		} else {
			for _, raw := range rawIndices {
				ix, err := strconv.ParseUint(raw, 10, 64)
				if err != nil {
					f.log.Error("could not parse index from request", "url", r.URL)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				indices = append(indices, int(ix))
			}
		}

		var mockBeaconBlockRoot [32]byte
		mockBeaconBlockRoot[0] = 42
		binary.LittleEndian.PutUint64(mockBeaconBlockRoot[32-8:], slot)
		sidecars := make([]*eth.BlobSidecar, len(indices))
		for i, ix := range indices {
			if ix < 0 || ix >= len(bundle.Blobs) {
				f.log.Error("blob index from request is out of range", "url", r.URL)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			sidecars[i] = &eth.BlobSidecar{
				BlockRoot:     mockBeaconBlockRoot,
				Slot:          eth.Uint64String(slot),
				Index:         eth.Uint64String(i),
				KZGCommitment: eth.Bytes48(bundle.Commitments[ix]),
				KZGProof:      eth.Bytes48(bundle.Proofs[ix]),
			}
			copy(sidecars[i].Blob[:], bundle.Blobs[ix])
		}
		if err := json.NewEncoder(w).Encode(&eth.APIGetBlobSidecarsResponse{Data: sidecars}); err != nil {
			f.log.Error("blobs handler err", "err", err)
		}
	})
	f.beaconSrv = &http.Server{
		Handler:           mux,
		ReadTimeout:       time.Second * 20,
		ReadHeaderTimeout: time.Second * 20,
		WriteTimeout:      time.Second * 20,
		IdleTimeout:       time.Second * 20,
	}
	go func() {
		if err := f.beaconSrv.Serve(f.beaconAPIListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			f.log.Error("failed to start fake-pos beacon server for blobs testing", "err", err)
		}
	}()
	return nil
}

func (f *FakeBeacon) StoreBlobsBundle(slot uint64, bundle *engine.BlobsBundleV1) error {
	data, err := json.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("failed to encode blobs bundle of slot %d: %w", slot, err)
	}

	f.blobsLock.Lock()
	defer f.blobsLock.Unlock()
	bundlePath := fmt.Sprintf("blobs_bundle_%d.json", slot)
	if err := os.MkdirAll(f.blobsDir, 0755); err != nil {
		return fmt.Errorf("failed to create dir for blob storage: %w", err)
	}
	err = os.WriteFile(filepath.Join(f.blobsDir, bundlePath), data, 0755)
	if err != nil {
		return fmt.Errorf("failed to write blobs bundle of slot %d: %w", slot, err)
	}
	return nil
}

func (f *FakeBeacon) LoadBlobsBundle(slot uint64) (*engine.BlobsBundleV1, error) {
	f.blobsLock.Lock()
	defer f.blobsLock.Unlock()
	bundlePath := fmt.Sprintf("blobs_bundle_%d.json", slot)
	data, err := os.ReadFile(filepath.Join(f.blobsDir, bundlePath))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("no blobs bundle found for slot %d (%q): %w", slot, bundlePath, ethereum.NotFound)
		} else {
			return nil, fmt.Errorf("failed to read blobs bundle of slot %d (%q): %w", slot, bundlePath, err)
		}
	}
	var out engine.BlobsBundleV1
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("failed to decode blobs bundle of slot %d (%q): %w", slot, bundlePath, err)
	}
	return &out, nil
}

func (f *FakeBeacon) Close() error {
	var out error
	if f.beaconSrv != nil {
		out = errors.Join(out, f.beaconSrv.Close())
	}
	if f.beaconAPIListener != nil {
		out = errors.Join(out, f.beaconAPIListener.Close())
	}
	return out
}

func (f *FakeBeacon) BeaconAddr() string {
	return "http://" + f.beaconAPIListener.Addr().String()
}

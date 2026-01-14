package fakebeacon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/blobstore"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
)

// FakeBeacon presents a beacon-node in testing, without leading any chain-building.
// This merely serves a fake beacon API, and holds on to blocks,
// to complement the actual block-building to happen in testing (e.g. through the fake consensus geth module).
type FakeBeacon struct {
	log log.Logger

	// in-memory blob store
	blobStore *blobstore.Store
	blobsLock sync.Mutex

	beaconSrv         *http.Server
	beaconAPIListener net.Listener

	fuluTime    *uint64
	genesisTime uint64
	blockTime   uint64
}

func NewBeacon(log log.Logger, blobStore *blobstore.Store, genesisTime uint64, blockTime uint64, fuluTime *uint64) *FakeBeacon {
	return &FakeBeacon{
		log:         log,
		blobStore:   blobStore,
		genesisTime: genesisTime,
		blockTime:   blockTime,
		fuluTime:    fuluTime,
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
	mux.HandleFunc("/eth/v1/beacon/blobs/", func(w http.ResponseWriter, r *http.Request) {
		if f.fuluTime == nil || time.Now().Before(time.Unix(int64(*f.fuluTime), 0)) {
			f.log.Warn("post-Fulu blobs endpoint queried before Fulu hardfork")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		blockID := strings.TrimPrefix(r.URL.Path, "/eth/v1/beacon/blobs/")
		slot, err := strconv.ParseUint(blockID, 10, 64)
		if err != nil {
			f.log.Error("could not parse block id from request", "url", r.URL.Path, "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bundle, err := f.LoadBlobsBundle(slot)
		if err != nil {
			f.log.Error("failed to load blobs bundle", "slot", slot, "err", err)
			if errors.Is(err, ethereum.NotFound) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		query := r.URL.Query()
		versionedHashes := make([]common.Hash, 0, len(bundle.Blobs))
		for _, raw := range query["versioned_hashes"] {
			versionedHashes = append(versionedHashes, common.HexToHash(raw))
		}
		blobs := make([]*eth.Blob, 0)
		for i := range bundle.Blobs {
			blob := eth.Blob(bundle.Blobs[i])
			commitment, err := kzg4844.BlobToCommitment(blob.KZGBlob())
			if err != nil {
				f.log.Error("could not compute blob commitment", "err", err)
			}
			versionedHash := eth.KZGToVersionedHash(kzg4844.Commitment(commitment))
			if len(versionedHashes) > 0 && !slices.Contains(versionedHashes, versionedHash) {
				continue
			}
			blobs = append(blobs, &blob)
		}

		if err := json.NewEncoder(w).Encode(&eth.APIBeaconBlobsResponse{Data: blobs}); err != nil {
			f.log.Error("blobs handler err", "err", err)
		}
	})
	mux.HandleFunc("/eth/v1/node/version", func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(&eth.APIVersionResponse{Data: eth.VersionInformation{Version: "fakebeacon 1.2.3"}})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			f.log.Error("version handler err", "err", err)
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

func (f *FakeBeacon) StoreBlobsBundle(slot uint64, bundle *engine.BlobsBundle) error {
	f.blobsLock.Lock()
	defer f.blobsLock.Unlock()

	// Solve for the slot timestamp.
	// slot = (timestamp - genesis) / slot_time
	// timestamp = slot * slot_time + genesis
	slotTimestamp := slot*f.blockTime + f.genesisTime

	for i, b := range bundle.Blobs {
		f.blobStore.StoreBlob(
			slotTimestamp,
			eth.IndexedBlobHash{
				Index: uint64(i),
				Hash:  eth.KZGToVersionedHash(kzg4844.Commitment(bundle.Commitments[i])),
			},
			(*eth.Blob)(b[:]),
		)
	}
	return nil
}

func (f *FakeBeacon) LoadBlobsBundle(slot uint64) (*engine.BlobsBundle, error) {
	f.blobsLock.Lock()
	defer f.blobsLock.Unlock()

	// Solve for the slot timestamp.
	// slot = (timestamp - genesis) / slot_time
	// timestamp = slot * slot_time + genesis
	slotTimestamp := slot*f.blockTime + f.genesisTime

	// Load blobs from the store
	// we wrap the slot timestamp in a l1 block ref to satisfy the getblobs API
	// TODO can we do this in a nicer way?
	ref := eth.L1BlockRef{
		Time: slotTimestamp,
	}
	blobs, err := f.blobStore.GetBlobs(context.Background(), ref, []eth.IndexedBlobHash{})
	if err != nil {
		return nil, fmt.Errorf("failed to load blobs from store: %w", err)
	}

	// Convert blobs to the bundle
	out := engine.BlobsBundle{
		// TODO if we don't use the other fields, better to return a different type
		Blobs: make([]hexutil.Bytes, len(blobs)),
	}
	for i, b := range blobs {
		out.Blobs[i] = hexutil.Bytes(b.String())
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

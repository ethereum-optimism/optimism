package datastore

import (
	"context"

	dsq "github.com/ipfs/go-datastore/query"
)

// NullDatastore stores nothing, but conforms to the API.
// Useful to test with.
type NullDatastore struct {
}

var _ Datastore = (*NullDatastore)(nil)
var _ Batching = (*NullDatastore)(nil)
var _ ScrubbedDatastore = (*NullDatastore)(nil)
var _ CheckedDatastore = (*NullDatastore)(nil)
var _ PersistentDatastore = (*NullDatastore)(nil)
var _ GCDatastore = (*NullDatastore)(nil)
var _ TxnDatastore = (*NullDatastore)(nil)

// NewNullDatastore constructs a null datastoe
func NewNullDatastore() *NullDatastore {
	return &NullDatastore{}
}

// Put implements Datastore.Put
func (d *NullDatastore) Put(ctx context.Context, key Key, value []byte) (err error) {
	return nil
}

// Sync implements Datastore.Sync
func (d *NullDatastore) Sync(ctx context.Context, prefix Key) error {
	return nil
}

// Get implements Datastore.Get
func (d *NullDatastore) Get(ctx context.Context, key Key) (value []byte, err error) {
	return nil, ErrNotFound
}

// Has implements Datastore.Has
func (d *NullDatastore) Has(ctx context.Context, key Key) (exists bool, err error) {
	return false, nil
}

// Has implements Datastore.GetSize
func (d *NullDatastore) GetSize(ctx context.Context, key Key) (size int, err error) {
	return -1, ErrNotFound
}

// Delete implements Datastore.Delete
func (d *NullDatastore) Delete(ctx context.Context, key Key) (err error) {
	return nil
}

func (d *NullDatastore) Scrub(ctx context.Context) error {
	return nil
}

func (d *NullDatastore) Check(ctx context.Context) error {
	return nil
}

// Query implements Datastore.Query
func (d *NullDatastore) Query(ctx context.Context, q dsq.Query) (dsq.Results, error) {
	return dsq.ResultsWithEntries(q, nil), nil
}

func (d *NullDatastore) Batch(ctx context.Context) (Batch, error) {
	return NewBasicBatch(d), nil
}

func (d *NullDatastore) CollectGarbage(ctx context.Context) error {
	return nil
}

func (d *NullDatastore) DiskUsage(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (d *NullDatastore) Close() error {
	return nil
}

func (d *NullDatastore) NewTransaction(ctx context.Context, readOnly bool) (Txn, error) {
	return &nullTxn{}, nil
}

type nullTxn struct{}

func (t *nullTxn) Get(ctx context.Context, key Key) (value []byte, err error) {
	return nil, nil
}

func (t *nullTxn) Has(ctx context.Context, key Key) (exists bool, err error) {
	return false, nil
}

func (t *nullTxn) GetSize(ctx context.Context, key Key) (size int, err error) {
	return 0, nil
}

func (t *nullTxn) Query(ctx context.Context, q dsq.Query) (dsq.Results, error) {
	return dsq.ResultsWithEntries(q, nil), nil
}

func (t *nullTxn) Put(ctx context.Context, key Key, value []byte) error {
	return nil
}

func (t *nullTxn) Delete(ctx context.Context, key Key) error {
	return nil
}

func (t *nullTxn) Commit(ctx context.Context) error {
	return nil
}

func (t *nullTxn) Discard(ctx context.Context) {}

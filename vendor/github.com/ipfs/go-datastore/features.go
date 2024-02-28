package datastore

import (
	"context"
	"reflect"
	"time"
)

const (
	FeatureNameBatching    = "Batching"
	FeatureNameChecked     = "Checked"
	FeatureNameGC          = "GC"
	FeatureNamePersistent  = "Persistent"
	FeatureNameScrubbed    = "Scrubbed"
	FeatureNameTTL         = "TTL"
	FeatureNameTransaction = "Transaction"
)

type BatchingFeature interface {
	Batch(ctx context.Context) (Batch, error)
}

type CheckedFeature interface {
	Check(ctx context.Context) error
}

type ScrubbedFeature interface {
	Scrub(ctx context.Context) error
}

type GCFeature interface {
	CollectGarbage(ctx context.Context) error
}

type PersistentFeature interface {
	// DiskUsage returns the space used by a datastore, in bytes.
	DiskUsage(ctx context.Context) (uint64, error)
}

// TTL encapulates the methods that deal with entries with time-to-live.
type TTL interface {
	PutWithTTL(ctx context.Context, key Key, value []byte, ttl time.Duration) error
	SetTTL(ctx context.Context, key Key, ttl time.Duration) error
	GetExpiration(ctx context.Context, key Key) (time.Time, error)
}

type TxnFeature interface {
	NewTransaction(ctx context.Context, readOnly bool) (Txn, error)
}

// Feature contains metadata about a datastore Feature.
type Feature struct {
	Name string
	// Interface is the nil interface of the feature.
	Interface interface{}
	// DatastoreInterface is the nil interface of the feature's corresponding datastore interface.
	DatastoreInterface interface{}
}

var featuresByName map[string]Feature

func init() {
	featuresByName = map[string]Feature{}
	for _, f := range Features() {
		featuresByName[f.Name] = f
	}
}

// Features returns a list of all known datastore features.
// This serves both to provide an authoritative list of features,
// and to define a canonical ordering of features.
func Features() []Feature {
	// for backwards compatibility, only append to this list
	return []Feature{
		{
			Name:               FeatureNameBatching,
			Interface:          (*BatchingFeature)(nil),
			DatastoreInterface: (*Batching)(nil),
		},
		{
			Name:               FeatureNameChecked,
			Interface:          (*CheckedFeature)(nil),
			DatastoreInterface: (*CheckedDatastore)(nil),
		},
		{
			Name:               FeatureNameGC,
			Interface:          (*GCFeature)(nil),
			DatastoreInterface: (*GCDatastore)(nil),
		},
		{
			Name:               FeatureNamePersistent,
			Interface:          (*PersistentFeature)(nil),
			DatastoreInterface: (*PersistentDatastore)(nil),
		},
		{
			Name:               FeatureNameScrubbed,
			Interface:          (*ScrubbedFeature)(nil),
			DatastoreInterface: (*ScrubbedDatastore)(nil),
		},
		{
			Name:               FeatureNameTTL,
			Interface:          (*TTL)(nil),
			DatastoreInterface: (*TTLDatastore)(nil),
		},
		{
			Name:               FeatureNameTransaction,
			Interface:          (*TxnFeature)(nil),
			DatastoreInterface: (*TxnDatastore)(nil),
		},
	}
}

// FeatureByName returns the feature with the given name, if known.
func FeatureByName(name string) (Feature, bool) {
	feat, known := featuresByName[name]
	return feat, known
}

// FeaturesForDatastore returns the features supported by the given datastore.
func FeaturesForDatastore(dstore Datastore) (features []Feature) {
	if dstore == nil {
		return nil
	}
	dstoreType := reflect.TypeOf(dstore)
	for _, f := range Features() {
		fType := reflect.TypeOf(f.Interface).Elem()
		if dstoreType.Implements(fType) {
			features = append(features, f)
		}
	}
	return
}

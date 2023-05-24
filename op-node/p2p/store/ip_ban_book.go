package store

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
	ds "github.com/ipfs/go-datastore"
)

const (
	ipBanCacheSize        = 100
	ipBanRecordExpiration = time.Hour * 24 * 7
)

var ipBanExpirationsBase = ds.NewKey("/ips/ban_expiration")

type ipBanRecord struct {
	Expiry     int64 `json:"expiry"`     // unix timestamp in seconds
	LastUpdate int64 `json:"lastUpdate"` // unix timestamp in seconds
}

func (s *ipBanRecord) SetLastUpdated(t time.Time) {
	s.LastUpdate = t.Unix()
}

func (s *ipBanRecord) LastUpdated() time.Time {
	return time.Unix(s.LastUpdate, 0)
}

func (s *ipBanRecord) MarshalBinary() (data []byte, err error) {
	return json.Marshal(s)
}

func (s *ipBanRecord) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}

type ipBanUpdate time.Time

func (p ipBanUpdate) Apply(rec *ipBanRecord) {
	rec.Expiry = time.Time(p).Unix()
}

type ipBanBook struct {
	book *recordsBook[string, *ipBanRecord]
}

func newIPBanRecord() *ipBanRecord {
	return new(ipBanRecord)
}

func ipKey(ip string) ds.Key {
	return ds.NewKey(ip)
}

func newIPBanBook(ctx context.Context, logger log.Logger, clock clock.Clock, store ds.Batching) (*ipBanBook, error) {
	book, err := newRecordsBook[string, *ipBanRecord](ctx, logger, clock, store, ipBanCacheSize, ipBanRecordExpiration, ipBanExpirationsBase, newIPBanRecord, ipKey)
	if err != nil {
		return nil, err
	}
	return &ipBanBook{book: book}, nil
}

func (d *ipBanBook) startGC() {
	d.book.startGC()
}

func (d *ipBanBook) GetIPBanExpiration(ip net.IP) (time.Time, error) {
	rec, err := d.book.getRecord(ip.To16().String())
	if err == UnknownRecordErr {
		return time.Time{}, UnknownBanErr
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(rec.Expiry, 0), nil
}

func (d *ipBanBook) SetIPBanExpiration(ip net.IP, expirationTime time.Time) error {
	if expirationTime == (time.Time{}) {
		return d.book.deleteRecord(ip.To16().String())
	}
	return d.book.SetRecord(ip.To16().String(), ipBanUpdate(expirationTime))
}

func (d *ipBanBook) Close() {
	d.book.Close()
}

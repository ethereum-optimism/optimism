package database

import (
	"errors"
	"fmt"
	"math/big"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/google/uuid"
)

/**
 * Types
 */

type BridgeMessage struct {
	MessageHash common.Hash `gorm:"primaryKey;serializer:bytes"`
	Nonce       *big.Int    `gorm:"serializer:u256"`

	SentMessageEventGUID    uuid.UUID
	RelayedMessageEventGUID *uuid.UUID

	Tx       Transaction `gorm:"embedded"`
	GasLimit *big.Int    `gorm:"serializer:u256"`
}

type L1BridgeMessage struct {
	BridgeMessage         `gorm:"embedded"`
	TransactionSourceHash common.Hash `gorm:"serializer:bytes"`
}

type L2BridgeMessage struct {
	BridgeMessage             `gorm:"embedded"`
	TransactionWithdrawalHash common.Hash `gorm:"serializer:bytes"`
}

type L2BridgeMessageVersionedMessageHash struct {
	MessageHash   common.Hash `gorm:"primaryKey;serializer:bytes"`
	V1MessageHash common.Hash `gorm:"serializer:bytes"`
}

type BridgeMessagesView interface {
	L1BridgeMessage(common.Hash) (*L1BridgeMessage, error)
	L1BridgeMessageWithFilter(BridgeMessage) (*L1BridgeMessage, error)

	L2BridgeMessage(common.Hash) (*L2BridgeMessage, error)
	L2BridgeMessageWithFilter(BridgeMessage) (*L2BridgeMessage, error)
}

type BridgeMessagesDB interface {
	BridgeMessagesView

	StoreL1BridgeMessages([]L1BridgeMessage) error
	MarkRelayedL1BridgeMessage(common.Hash, uuid.UUID) error

	StoreL2BridgeMessages([]L2BridgeMessage) error
	MarkRelayedL2BridgeMessage(common.Hash, uuid.UUID) error

	StoreL2BridgeMessageV1MessageHashes([]L2BridgeMessageVersionedMessageHash) error
}

/**
 * Implementation
 */

type bridgeMessagesDB struct {
	log  log.Logger
	gorm *gorm.DB
}

func newBridgeMessagesDB(log log.Logger, db *gorm.DB) BridgeMessagesDB {
	return &bridgeMessagesDB{log: log.New("table", "bridge_messages"), gorm: db}
}

/**
 * Arbitrary Messages Sent from L1
 */

func (db bridgeMessagesDB) StoreL1BridgeMessages(messages []L1BridgeMessage) error {
	deduped := db.gorm.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "message_hash"}}, DoNothing: true})
	result := deduped.Create(&messages)
	if result.Error == nil && int(result.RowsAffected) < len(messages) {
		db.log.Warn("ignored L1 bridge message duplicates", "duplicates", len(messages)-int(result.RowsAffected))
	}

	return result.Error
}

func (db bridgeMessagesDB) L1BridgeMessage(msgHash common.Hash) (*L1BridgeMessage, error) {
	return db.L1BridgeMessageWithFilter(BridgeMessage{MessageHash: msgHash})
}

func (db bridgeMessagesDB) L1BridgeMessageWithFilter(filter BridgeMessage) (*L1BridgeMessage, error) {
	var sentMessage L1BridgeMessage
	result := db.gorm.Where(&filter).Take(&sentMessage)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &sentMessage, nil
}

func (db bridgeMessagesDB) MarkRelayedL1BridgeMessage(messageHash common.Hash, relayEvent uuid.UUID) error {
	message, err := db.L1BridgeMessage(messageHash)
	if err != nil {
		return err
	} else if message == nil {
		return fmt.Errorf("L1BridgeMessage %s not found", messageHash)
	}

	if message.RelayedMessageEventGUID != nil && message.RelayedMessageEventGUID.ID() == relayEvent.ID() {
		return nil
	} else if message.RelayedMessageEventGUID != nil {
		return fmt.Errorf("relayed message %s re-relayed with a different event %d", messageHash, relayEvent)
	}

	message.RelayedMessageEventGUID = &relayEvent
	result := db.gorm.Save(message)
	return result.Error
}

/**
 * Arbitrary Messages Sent from L2
 */

func (db bridgeMessagesDB) StoreL2BridgeMessages(messages []L2BridgeMessage) error {
	deduped := db.gorm.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "message_hash"}}, DoNothing: true})
	result := deduped.Create(&messages)
	if result.Error == nil && int(result.RowsAffected) < len(messages) {
		db.log.Warn("ignored L2 bridge message duplicates", "duplicates", len(messages)-int(result.RowsAffected))
	}

	return result.Error
}

func (db bridgeMessagesDB) StoreL2BridgeMessageV1MessageHashes(versionedHashes []L2BridgeMessageVersionedMessageHash) error {
	deduped := db.gorm.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "message_hash"}}, DoNothing: true})
	result := deduped.Create(&versionedHashes)
	if result.Error == nil && int(result.RowsAffected) < len(versionedHashes) {
		db.log.Warn("ignored L2 bridge v1 message hash duplicates", "duplicates", len(versionedHashes)-int(result.RowsAffected))
	}

	return result.Error
}

func (db bridgeMessagesDB) L2BridgeMessage(msgHash common.Hash) (*L2BridgeMessage, error) {
	message, err := db.L2BridgeMessageWithFilter(BridgeMessage{MessageHash: msgHash})
	if message != nil || err != nil {
		return message, err
	}

	// check if this is a v1 hash of an older message
	versioned := L2BridgeMessageVersionedMessageHash{V1MessageHash: msgHash}
	result := db.gorm.Where(&versioned).Take(&versioned)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return db.L2BridgeMessageWithFilter(BridgeMessage{MessageHash: versioned.MessageHash})
}

func (db bridgeMessagesDB) L2BridgeMessageWithFilter(filter BridgeMessage) (*L2BridgeMessage, error) {
	var sentMessage L2BridgeMessage
	result := db.gorm.Where(&filter).Take(&sentMessage)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &sentMessage, nil
}

func (db bridgeMessagesDB) MarkRelayedL2BridgeMessage(messageHash common.Hash, relayEvent uuid.UUID) error {
	message, err := db.L2BridgeMessage(messageHash)
	if err != nil {
		return err
	} else if message == nil {
		return fmt.Errorf("L2BridgeMessage %s not found", messageHash)
	}

	if message.RelayedMessageEventGUID != nil && message.RelayedMessageEventGUID.ID() == relayEvent.ID() {
		return nil
	} else if message.RelayedMessageEventGUID != nil {
		return fmt.Errorf("relayed message %s re-relayed with a different event %s", messageHash, relayEvent)
	}

	message.RelayedMessageEventGUID = &relayEvent
	result := db.gorm.Save(message)
	return result.Error
}
